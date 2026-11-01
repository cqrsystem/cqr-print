//go:build windows

package impressao

import (
	"errors"
	"fmt"
	"image"
	"math"
	"syscall"
	"unsafe"

	"github.com/cqrsystem/cqr-print/internal/api"
)

var (
	winspool              = syscall.NewLazyDLL("winspool.drv")
	gdi32                 = syscall.NewLazyDLL("gdi32.dll")
	procEnumPrinters      = winspool.NewProc("EnumPrintersW")
	procGetDefaultPrinter = winspool.NewProc("GetDefaultPrinterW")
	procOpenPrinter       = winspool.NewProc("OpenPrinterW")
	procClosePrinter      = winspool.NewProc("ClosePrinter")
	procStartDocPrinter   = winspool.NewProc("StartDocPrinterW")
	procEndDocPrinter     = winspool.NewProc("EndDocPrinter")
	procStartPagePrinter  = winspool.NewProc("StartPagePrinter")
	procEndPagePrinter    = winspool.NewProc("EndPagePrinter")
	procWritePrinter      = winspool.NewProc("WritePrinter")
	procCreateDC          = gdi32.NewProc("CreateDCW")
	procDeleteDC          = gdi32.NewProc("DeleteDC")
	procStartDoc          = gdi32.NewProc("StartDocW")
	procEndDoc            = gdi32.NewProc("EndDoc")
	procAbortDoc          = gdi32.NewProc("AbortDoc")
	procStartPage         = gdi32.NewProc("StartPage")
	procEndPage           = gdi32.NewProc("EndPage")
	procGetDeviceCaps     = gdi32.NewProc("GetDeviceCaps")
	procStretchDIBits     = gdi32.NewProc("StretchDIBits")
	procSetStretchBltMode = gdi32.NewProc("SetStretchBltMode")
)

const (
	enumLocal       = 0x2
	enumConexoes    = 0x4
	logPixelsX      = 88
	logPixelsY      = 90
	dibRGB          = 0
	srcCopy         = 0x00CC0020
	colorOnColor    = 3
	errInsuficiente = 122 // ERROR_INSUFFICIENT_BUFFER
)

type printerInfo4 struct {
	nome      *uint16
	servidor  *uint16
	atributos uint32
}

func listar() ([]api.ImpressoraDoWindows, error) {
	var precisa, total uint32
	flags := uintptr(enumLocal | enumConexoes)
	procEnumPrinters.Call(flags, 0, 4, 0, 0, uintptr(unsafe.Pointer(&precisa)), uintptr(unsafe.Pointer(&total)))
	if precisa == 0 {
		return nil, nil
	}
	buf := make([]byte, precisa)
	r, _, err := procEnumPrinters.Call(flags, 0, 4, uintptr(unsafe.Pointer(&buf[0])), uintptr(precisa),
		uintptr(unsafe.Pointer(&precisa)), uintptr(unsafe.Pointer(&total)))
	if r == 0 {
		return nil, fmt.Errorf("não listou as impressoras: %w", err)
	}
	padrao := impressoraPadrao()
	infos := unsafe.Slice((*printerInfo4)(unsafe.Pointer(&buf[0])), total)
	lista := make([]api.ImpressoraDoWindows, 0, total)
	for _, i := range infos {
		nome := syscall.UTF16ToString(unsafe.Slice(i.nome, comprimento(i.nome)))
		lista = append(lista, api.ImpressoraDoWindows{Nome: nome, Padrao: nome == padrao})
	}
	return lista, nil
}

func comprimento(p *uint16) int {
	if p == nil {
		return 0
	}
	n := 0
	for *(*uint16)(unsafe.Add(unsafe.Pointer(p), n*2)) != 0 {
		n++
	}
	return n
}

func impressoraPadrao() string {
	var n uint32
	procGetDefaultPrinter.Call(0, uintptr(unsafe.Pointer(&n)))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n)
	r, _, _ := procGetDefaultPrinter.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

type docInfo1 struct {
	nome  *uint16
	saida *uint16
	tipo  *uint16
}

// enviarRaw manda os bytes (ZPL) para a impressora do Windows sem passar pelo desenho do driver.
func enviarRaw(impressora, documento string, dados []byte) error {
	if impressora == "" {
		return errors.New("impressora do Windows não informada")
	}
	nome, err := syscall.UTF16PtrFromString(impressora)
	if err != nil {
		return err
	}
	var h syscall.Handle
	if r, _, err := procOpenPrinter.Call(uintptr(unsafe.Pointer(nome)), uintptr(unsafe.Pointer(&h)), 0); r == 0 {
		return fmt.Errorf("não abriu a impressora %q: %w", impressora, err)
	}
	defer procClosePrinter.Call(uintptr(h))

	doc, _ := syscall.UTF16PtrFromString(nomeDocumento(documento))
	raw, _ := syscall.UTF16PtrFromString("RAW")
	info := docInfo1{nome: doc, tipo: raw}
	if r, _, err := procStartDocPrinter.Call(uintptr(h), 1, uintptr(unsafe.Pointer(&info))); r == 0 {
		return fmt.Errorf("a impressora recusou o trabalho: %w", err)
	}
	defer procEndDocPrinter.Call(uintptr(h))
	procStartPagePrinter.Call(uintptr(h))
	defer procEndPagePrinter.Call(uintptr(h))
	var escrito uint32
	if r, _, err := procWritePrinter.Call(uintptr(h), uintptr(unsafe.Pointer(&dados[0])), uintptr(len(dados)), uintptr(unsafe.Pointer(&escrito))); r == 0 {
		return fmt.Errorf("falha ao enviar para a impressora: %w", err)
	}
	if int(escrito) != len(dados) {
		return fmt.Errorf("a impressora recebeu %d de %d bytes", escrito, len(dados))
	}
	return nil
}

func nomeDocumento(d string) string {
	if d == "" {
		return "Etiqueta CQR"
	}
	return "CQR - " + d
}

type docInfoW struct {
	tamanho int32
	nome    *uint16
	saida   *uint16
	tipo    *uint16
	flags   uint32
}

type bitmapInfoHeader struct {
	tamanho     uint32
	largura     int32
	altura      int32
	planos      uint16
	bits        uint16
	compressao  uint32
	tamImagem   uint32
	xPorMetro   int32
	yPorMetro   int32
	coresUsadas uint32
	coresImport uint32
}

// imprimirImagem desenha a etiqueta pelo driver do Windows, no tamanho em mm, uma página por cópia.
// O tamanho do papel (etiqueta) é o configurado no driver da impressora.
func imprimirImagem(impressora, documento string, img image.Image, larguraMM, alturaMM float64, copias int) error {
	if impressora == "" {
		return errors.New("impressora do Windows não informada")
	}
	nome, err := syscall.UTF16PtrFromString(impressora)
	if err != nil {
		return err
	}
	hdc, _, err := procCreateDC.Call(0, uintptr(unsafe.Pointer(nome)), 0, 0)
	if hdc == 0 {
		return fmt.Errorf("não abriu a impressora %q: %w", impressora, err)
	}
	defer procDeleteDC.Call(hdc)

	dpiX, _, _ := procGetDeviceCaps.Call(hdc, logPixelsX)
	dpiY, _, _ := procGetDeviceCaps.Call(hdc, logPixelsY)
	pixels, w, h := PixelsBGR(img)
	destW, destH := int(math.Round(larguraMM/25.4*float64(dpiX))), int(math.Round(alturaMM/25.4*float64(dpiY)))
	if larguraMM <= 0 || alturaMM <= 0 {
		destW, destH = w, h
	}
	cabecalho := bitmapInfoHeader{tamanho: 40, largura: int32(w), altura: -int32(h), planos: 1, bits: 24}

	doc, _ := syscall.UTF16PtrFromString(nomeDocumento(documento))
	info := docInfoW{tamanho: int32(unsafe.Sizeof(docInfoW{})), nome: doc}
	if r, _, err := procStartDoc.Call(hdc, uintptr(unsafe.Pointer(&info))); int32(r) <= 0 {
		return fmt.Errorf("a impressora recusou o trabalho: %w", err)
	}
	procSetStretchBltMode.Call(hdc, colorOnColor)
	for i := 0; i < copias; i++ {
		if r, _, err := procStartPage.Call(hdc); int32(r) <= 0 {
			procAbortDoc.Call(hdc)
			return fmt.Errorf("falha ao começar a página: %w", err)
		}
		r, _, err := procStretchDIBits.Call(hdc, 0, 0, uintptr(destW), uintptr(destH), 0, 0, uintptr(w), uintptr(h),
			uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&cabecalho)), dibRGB, srcCopy)
		if r == 0 {
			procAbortDoc.Call(hdc)
			return fmt.Errorf("falha ao desenhar a etiqueta: %w", err)
		}
		procEndPage.Call(hdc)
	}
	if r, _, err := procEndDoc.Call(hdc); int32(r) <= 0 {
		return fmt.Errorf("falha ao terminar o trabalho: %w", err)
	}
	return nil
}
