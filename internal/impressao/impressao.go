// Package impressao entrega os trabalhos às impressoras:
//   - ZPL (Zebra): em RAW pelo spooler do Windows, ou direto na porta 9100 do IP;
//   - imagem (Argox, laser, jato de tinta): o PNG pelo driver do Windows (GDI), no tamanho da etiqueta.
package impressao

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/cqrsystem/cqr-print/internal/api"
)

// Impressoras do computador e a entrega de cada trabalho. No Windows é a implementação real; nos
// testes, uma falsa.
type Impressoras interface {
	Listar() ([]api.ImpressoraDoWindows, error)
	Imprimir(t api.Trabalho) error
}

// Sistema: as impressoras deste computador.
type Sistema struct{}

// Listar devolve as impressoras instaladas no Windows (vazio fora do Windows).
func (Sistema) Listar() ([]api.ImpressoraDoWindows, error) { return listar() }

// Imprimir entrega um trabalho conforme a impressora e a linguagem.
func (Sistema) Imprimir(t api.Trabalho) error {
	copias := t.Copias
	if copias < 1 {
		copias = 1
	}
	switch {
	case t.Linguagem == "zpl" || t.Formato == "texto":
		// O ZPL já traz as cópias (^PQ).
		dados := []byte(t.Conteudo)
		if t.Impressora.Destino == "rede" {
			return EnviarRede(t.Impressora.Endereco, t.Impressora.Porta, dados)
		}
		return enviarRaw(t.Impressora.NomeWindows, t.Descricao, dados)
	case t.Formato == "png_base64":
		img, err := DecodificarPNG(t.Conteudo)
		if err != nil {
			return err
		}
		if t.Impressora.Destino == "rede" {
			return errors.New("impressora em rede (IP) só recebe ZPL; cadastre-a como impressora do Windows")
		}
		return imprimirImagem(t.Impressora.NomeWindows, t.Descricao, img, t.LarguraMM, t.AlturaMM, copias)
	default:
		return fmt.Errorf("formato desconhecido: %s", t.Formato)
	}
}

// EnviarRede manda os comandos direto para a impressora na porta (9100).
func EnviarRede(endereco string, porta int, dados []byte) error {
	if porta <= 0 {
		porta = 9100
	}
	if strings.TrimSpace(endereco) == "" {
		return errors.New("impressora sem endereço (IP)")
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(endereco, strconv.Itoa(porta)), 10*time.Second)
	if err != nil {
		return fmt.Errorf("não conectou na impressora %s:%d: %w", endereco, porta, err)
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	_, err = conn.Write(dados)
	return err
}

// DecodificarPNG abre a imagem que veio em base64.
func DecodificarPNG(b64 string) (image.Image, error) {
	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("imagem inválida: %w", err)
	}
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("imagem inválida: %w", err)
	}
	return img, nil
}

// PixelsBGR converte a imagem para linhas BGR de 24 bits, de cima para baixo, cada linha com
// múltiplo de 4 bytes (o formato DIB que o Windows imprime).
func PixelsBGR(img image.Image) (dados []byte, largura, altura int) {
	r := img.Bounds()
	largura, altura = r.Dx(), r.Dy()
	passo := (largura*3 + 3) &^ 3
	dados = make([]byte, passo*altura)
	for y := 0; y < altura; y++ {
		linha := dados[y*passo:]
		for x := 0; x < largura; x++ {
			cr, cg, cb, _ := img.At(r.Min.X+x, r.Min.Y+y).RGBA()
			linha[x*3] = byte(cb >> 8)
			linha[x*3+1] = byte(cg >> 8)
			linha[x*3+2] = byte(cr >> 8)
		}
	}
	return dados, largura, altura
}
