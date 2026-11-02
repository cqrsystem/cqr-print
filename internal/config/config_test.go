package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSalvarCarregarApagar(t *testing.T) {
	t.Setenv("CQRPRINT_PASTA", t.TempDir())
	if _, err := Carregar(); !errors.Is(err, ErrSemConfig) {
		t.Fatalf("sem config: %v", err)
	}
	c := &Config{Servidor: "https://x", Nome: "PC", Empresa: "ACME", Token: "1|segredo"}
	if err := c.Salvar(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(arquivo())
	if strings.Contains(string(b), "1|segredo") {
		t.Fatal("o token não pode ficar aberto no arquivo")
	}
	lido, err := Carregar()
	if err != nil || lido.Token != "1|segredo" || lido.Empresa != "ACME" {
		t.Fatalf("carregar: %v %+v", err, lido)
	}
	if err := Apagar(); err != nil {
		t.Fatal(err)
	}
	if _, err := Carregar(); !errors.Is(err, ErrSemConfig) {
		t.Fatalf("depois de apagar: %v", err)
	}
}
