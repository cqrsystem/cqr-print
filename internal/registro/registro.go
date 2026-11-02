// Package registro escreve o registro do agente num arquivo (com troca ao passar de 5 MB) e na tela.
package registro

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const limite = 5 << 20

type arquivo struct {
	mu      sync.Mutex
	caminho string
	f       *os.File
	tamanho int64
	tela    io.Writer
}

func (a *arquivo) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tela != nil {
		_, _ = a.tela.Write(p)
	}
	if a.f == nil || a.tamanho+int64(len(p)) > limite {
		a.trocar()
	}
	if a.f == nil {
		return len(p), nil
	}
	n, err := a.f.Write(p)
	a.tamanho += int64(n)
	return n, err
}

func (a *arquivo) trocar() {
	if a.f != nil {
		_ = a.f.Close()
		_ = os.Rename(a.caminho, a.caminho+".1")
	}
	f, err := os.OpenFile(a.caminho, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		a.f = nil
		return
	}
	st, _ := f.Stat()
	a.f, a.tamanho = f, 0
	if st != nil {
		a.tamanho = st.Size()
	}
}

// Iniciar liga o registro na pasta do agente; com tela, escreve também na janela.
func Iniciar(pasta string, tela bool) {
	_ = os.MkdirAll(pasta, 0o700)
	a := &arquivo{caminho: filepath.Join(pasta, "cqrprint.log")}
	if tela {
		a.tela = os.Stdout
	}
	log.SetOutput(a)
	log.SetFlags(log.LstdFlags)
}

// Linha escreve uma linha no registro.
func Linha(formato string, args ...any) {
	log.Print(fmt.Sprintf(formato, args...))
}
