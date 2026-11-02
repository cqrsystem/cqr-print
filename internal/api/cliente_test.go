package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAtivarSinalTrabalhosResultados(t *testing.T) {
	var corpoAtivar, corpoResultados map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/v1/aparelhos/ativar" && r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"erro":{"codigo":"nao_autenticado","mensagem":"Aparelho não reconhecido"}}`))
			return
		}
		switch r.URL.Path {
		case "/api/v1/aparelhos/ativar":
			_ = json.NewDecoder(r.Body).Decode(&corpoAtivar)
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"dados":{"token":"tok","aparelho":{"id":7,"nome":"PC 1"},"empresa":{"nome":"ACME"}},"meta":{}}`))
		case "/api/v1/impressao/sinal":
			_, _ = w.Write([]byte(`{"dados":{"empresa":"ACME","intervalo_seg":3,"impressoras":[{"id":1,"codigo":"ZEBRA1","destino":"windows","nome_windows":"ZD220","linguagem":"zpl","dpi":203}]},"meta":{}}`))
		case "/api/v1/impressao/trabalhos":
			if r.URL.Query().Get("limite") != "5" {
				t.Errorf("limite = %s", r.URL.Query().Get("limite"))
			}
			_, _ = w.Write([]byte(`{"dados":[{"id":10,"impressora":{"id":1,"codigo":"ZEBRA1","destino":"windows","nome_windows":"ZD220","linguagem":"zpl"},"descricao":"Série S1","linguagem":"zpl","conteudo":"^XA^XZ","formato":"texto","copias":2,"largura_mm":50,"altura_mm":25,"dpi":203}],"meta":{"intervalo_seg":3}}`))
		case "/api/v1/impressao/resultados":
			_ = json.NewDecoder(r.Body).Decode(&corpoResultados)
			_, _ = w.Write([]byte(`{"dados":{"aceitos":[10]},"meta":{}}`))
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	c := Novo(srv.URL+"/", "", "1.0")
	a, err := c.Ativar(ctx, "99.888.777/0001-00", "1001", "segredo", "PC 1")
	if err != nil || a.Token != "tok" || a.Empresa.Nome != "ACME" {
		t.Fatalf("ativar: %v %+v", err, a)
	}
	if corpoAtivar["tipo"] != "impressao" || corpoAtivar["matricula"] != "1001" {
		t.Fatalf("corpo da ativação: %v", corpoAtivar)
	}

	if _, err := c.EnviarSinal(ctx, "1.0", nil); !SemAcesso(err) {
		t.Fatalf("sem token deveria ser recusado: %v", err)
	}
	c.Token = "tok"
	s, err := c.EnviarSinal(ctx, "1.0", []ImpressoraDoWindows{{Nome: "ZD220", Padrao: true}})
	if err != nil || len(s.Impressoras) != 1 || s.Impressoras[0].NomeWindows != "ZD220" {
		t.Fatalf("sinal: %v %+v", err, s)
	}
	tr, err := c.BuscarTrabalhos(ctx, 5)
	if err != nil || len(tr) != 1 || tr[0].Copias != 2 || tr[0].Conteudo != "^XA^XZ" {
		t.Fatalf("trabalhos: %v %+v", err, tr)
	}
	aceitos, err := c.EnviarResultados(ctx, []Resultado{{ID: 10, OK: false, Erro: "Sem papel"}})
	if err != nil || len(aceitos) != 1 {
		t.Fatalf("resultados: %v %v", err, aceitos)
	}
	r := corpoResultados["resultados"].([]any)[0].(map[string]any)
	if r["ok"] != false || r["erro"] != "Sem papel" {
		t.Fatalf("corpo dos resultados: %v", corpoResultados)
	}
}

func TestValidarServidor(t *testing.T) {
	for _, ok := range []string{"https://cqr.exemplo.com.br", "http://localhost:8000"} {
		if err := ValidarServidor(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, ruim := range []string{"cqr.exemplo.com.br", "http://cqr.exemplo.com.br", "ftp://x", ""} {
		if ValidarServidor(ruim) == nil {
			t.Errorf("%s deveria ser recusado", ruim)
		}
	}
}
