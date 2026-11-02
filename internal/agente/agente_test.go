package agente

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cqrsystem/cqr-print/internal/api"
)

type servidorFalso struct {
	filas      [][]api.Trabalho
	resultados [][]api.Resultado
	sinais     int
	falhas     int
	revogado   bool
}

func (s *servidorFalso) EnviarSinal(context.Context, string, []api.ImpressoraDoWindows) (*api.Sinal, error) {
	s.sinais++
	if s.revogado {
		return nil, &api.Erro{Status: 401, Codigo: "aparelho_revogado"}
	}
	return &api.Sinal{Empresa: "ACME", IntervaloSeg: 3}, nil
}

func (s *servidorFalso) BuscarTrabalhos(context.Context, int) ([]api.Trabalho, error) {
	if s.falhas > 0 {
		s.falhas--
		return nil, errors.New("rede caiu")
	}
	if len(s.filas) == 0 {
		return nil, nil
	}
	t := s.filas[0]
	s.filas = s.filas[1:]
	return t, nil
}

func (s *servidorFalso) EnviarResultados(_ context.Context, r []api.Resultado) ([]int, error) {
	s.resultados = append(s.resultados, r)
	return nil, nil
}

type impressorasFalsas struct{ impressos []int }

func (i *impressorasFalsas) Listar() ([]api.ImpressoraDoWindows, error) {
	return []api.ImpressoraDoWindows{{Nome: "ZD220"}}, nil
}

func (i *impressorasFalsas) Imprimir(t api.Trabalho) error {
	if t.Impressora.Codigo == "SEMPAPEL" {
		return errors.New("Sem papel")
	}
	i.impressos = append(i.impressos, t.ID)
	return nil
}

func TestRodaImprimeResponde(t *testing.T) {
	s := &servidorFalso{falhas: 1, filas: [][]api.Trabalho{
		{{ID: 1, Impressora: api.Impressora{Codigo: "Z1"}}, {ID: 2, Impressora: api.Impressora{Codigo: "SEMPAPEL"}}},
		{{ID: 3, Impressora: api.Impressora{Codigo: "Z1"}}},
	}}
	imp := &impressorasFalsas{}
	a := Novo(s, imp, "teste")
	esperas := 0
	ctx, cancelar := context.WithCancel(context.Background())
	a.esperar = func(_ context.Context, d time.Duration) bool {
		esperas++
		if len(s.filas) == 0 && s.falhas == 0 && esperas > 3 {
			cancelar()
			return false
		}
		return true
	}
	if err := a.Rodar(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Rodar: %v", err)
	}
	if len(imp.impressos) != 2 || imp.impressos[0] != 1 || imp.impressos[1] != 3 {
		t.Fatalf("impressos: %v", imp.impressos)
	}
	if len(s.resultados) != 2 || s.resultados[0][1].OK || s.resultados[0][1].Erro != "Sem papel" || !s.resultados[0][0].OK {
		t.Fatalf("resultados: %+v", s.resultados)
	}
	if s.sinais != 1 {
		t.Fatalf("sinais: %d", s.sinais)
	}
}

func TestAgenteRevogadoPara(t *testing.T) {
	a := Novo(&servidorFalso{revogado: true}, &impressorasFalsas{}, "teste")
	a.esperar = func(context.Context, time.Duration) bool { return true }
	if err := a.Rodar(context.Background()); !errors.Is(err, ErrDesativado) {
		t.Fatalf("esperava ErrDesativado, veio %v", err)
	}
}
