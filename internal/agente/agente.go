// Package agente é o laço do CQR Print: apresenta-se ao servidor (sinal), busca os trabalhos,
// imprime e devolve os resultados em lote (docs/api.md 4.2 do cqr-system-web).
package agente

import (
	"context"
	"errors"
	"time"

	"github.com/cqrsystem/cqr-print/internal/api"
	"github.com/cqrsystem/cqr-print/internal/impressao"
	"github.com/cqrsystem/cqr-print/internal/registro"
)

// Servidor: o que o agente usa da API (o cliente real ou um falso nos testes).
type Servidor interface {
	EnviarSinal(ctx context.Context, versao string, impressoras []api.ImpressoraDoWindows) (*api.Sinal, error)
	BuscarTrabalhos(ctx context.Context, limite int) ([]api.Trabalho, error)
	EnviarResultados(ctx context.Context, resultados []api.Resultado) ([]int, error)
}

// ErrDesativado: o servidor não reconhece mais este agente (revogado no Desk). É preciso ativar de novo.
var ErrDesativado = errors.New("este agente foi desativado no CQR System; ative de novo")

// Agente de impressão.
type Agente struct {
	Servidor    Servidor
	Impressoras impressao.Impressoras
	Versao      string
	// SinalCada: de quanto em quanto tempo reenviar a lista de impressoras do Windows.
	SinalCada time.Duration
	Limite    int
	// esperar permite acelerar os testes.
	esperar func(ctx context.Context, d time.Duration) bool
}

// Novo cria o agente com os padrões.
func Novo(s Servidor, i impressao.Impressoras, versao string) *Agente {
	return &Agente{Servidor: s, Impressoras: i, Versao: versao, SinalCada: 5 * time.Minute, Limite: 10, esperar: esperarCtx}
}

func esperarCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// Rodar fica no laço até o contexto acabar ou o agente ser desativado. Falha de rede não para o
// agente: espera mais um pouco (até 1 minuto) e tenta de novo.
func (a *Agente) Rodar(ctx context.Context) error {
	intervalo := 3 * time.Second
	espera := intervalo
	var ultimoSinal time.Time
	for {
		if ultimoSinal.IsZero() || time.Since(ultimoSinal) >= a.SinalCada {
			s, err := a.sinal(ctx)
			if err == nil {
				ultimoSinal = time.Now()
				if s.IntervaloSeg > 0 {
					intervalo = time.Duration(s.IntervaloSeg) * time.Second
				}
			} else if api.SemAcesso(err) {
				return ErrDesativado
			}
		}

		n, err := a.Rodada(ctx)
		switch {
		case api.SemAcesso(err):
			return ErrDesativado
		case err != nil:
			registro.Linha("sem contato com o servidor: %v", err)
			espera = min(espera*2, time.Minute)
		case n > 0:
			espera = 0 // pode haver mais na fila
		default:
			espera = intervalo
		}
		if espera > 0 && !a.esperar(ctx, espera) {
			return ctx.Err()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (a *Agente) sinal(ctx context.Context) (*api.Sinal, error) {
	lista, err := a.Impressoras.Listar()
	if err != nil {
		registro.Linha("não listou as impressoras do Windows: %v", err)
	}
	s, err := a.Servidor.EnviarSinal(ctx, a.Versao, lista)
	if err != nil {
		registro.Linha("sinal não enviado: %v", err)
		return nil, err
	}
	registro.Linha("conectado a %s: %d impressora(s) configurada(s), %d no Windows", s.Empresa, len(s.Impressoras), len(lista))
	return s, nil
}

// Rodada busca os trabalhos, imprime cada um e devolve os resultados. Devolve quantos imprimiu
// (ou tentou).
func (a *Agente) Rodada(ctx context.Context) (int, error) {
	trabalhos, err := a.Servidor.BuscarTrabalhos(ctx, a.Limite)
	if err != nil || len(trabalhos) == 0 {
		return 0, err
	}
	resultados := make([]api.Resultado, 0, len(trabalhos))
	for _, t := range trabalhos {
		r := api.Resultado{ID: t.ID, OK: true}
		if err := a.Impressoras.Imprimir(t); err != nil {
			r.OK, r.Erro = false, limitar(err.Error(), 250)
			registro.Linha("erro ao imprimir %q na %s: %v", t.Descricao, t.Impressora.Codigo, err)
		} else {
			registro.Linha("impresso: %s na %s", t.Descricao, t.Impressora.Codigo)
		}
		resultados = append(resultados, r)
	}
	// O resultado não pode se perder: tenta algumas vezes; sem resposta, o servidor reenvia depois.
	for tentativa := 0; tentativa < 3; tentativa++ {
		if _, err = a.Servidor.EnviarResultados(ctx, resultados); err == nil || api.SemAcesso(err) {
			break
		}
		if !a.esperar(ctx, time.Duration(tentativa+1)*2*time.Second) {
			break
		}
	}
	return len(trabalhos), err
}

func limitar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
