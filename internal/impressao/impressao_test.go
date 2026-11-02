package impressao

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"testing"

	"github.com/cqrsystem/cqr-print/internal/api"
)

func TestPixelsBGRComLinhasMultiplasDe4(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(2, 1, color.RGBA{B: 255, A: 255})
	dados, w, h := PixelsBGR(img)
	if w != 3 || h != 2 || len(dados) != 24 { // 3*3=9 -> 12 por linha
		t.Fatalf("w=%d h=%d len=%d", w, h, len(dados))
	}
	if dados[2] != 255 || dados[0] != 0 { // vermelho em BGR
		t.Fatalf("pixel 0,0: %v", dados[:3])
	}
	if dados[12+6] != 255 { // azul em x=2 da linha 1
		t.Fatalf("pixel 2,1: %v", dados[12+6:12+9])
	}
}

func TestEnviarRedeEPng(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	recebido := make(chan []byte, 1)
	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		b, _ := io.ReadAll(c)
		recebido <- b
	}()
	porta := l.Addr().(*net.TCPAddr).Port
	err = Sistema{}.Imprimir(api.Trabalho{Linguagem: "zpl", Formato: "texto", Conteudo: "^XA^FDoi^FS^XZ",
		Impressora: api.Impressora{Destino: "rede", Endereco: "127.0.0.1", Porta: porta}})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(<-recebido); got != "^XA^FDoi^FS^XZ" {
		t.Fatalf("recebido: %q", got)
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewGray(image.Rect(0, 0, 4, 2)))
	img, err := DecodificarPNG(base64.StdEncoding.EncodeToString(buf.Bytes()))
	if err != nil || img.Bounds().Dx() != 4 {
		t.Fatalf("png: %v", err)
	}
	if err := (Sistema{}).Imprimir(api.Trabalho{Formato: "png_base64", Conteudo: base64.StdEncoding.EncodeToString(buf.Bytes()),
		Impressora: api.Impressora{Destino: "rede", Endereco: "127.0.0.1"}}); err == nil {
		t.Fatal("imagem em impressora de rede deveria ser recusada")
	}
}
