package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrocarArquivo(t *testing.T) {
	pasta := t.TempDir()
	destino := filepath.Join(pasta, "cqrprint.exe")
	novo := destino + ".novo"
	ler := func(p string) string { b, _ := os.ReadFile(p); return string(b) }

	// Primeira instalação: não havia programa.
	os.WriteFile(novo, []byte("v1"), 0o644)
	if err := trocarArquivo(novo, destino); err != nil || ler(destino) != "v1" {
		t.Fatalf("instalação: %v %q", err, ler(destino))
	}
	// Atualização: o novo fica no lugar e não sobra o .novo.
	os.WriteFile(novo, []byte("v2"), 0o644)
	if err := trocarArquivo(novo, destino); err != nil || ler(destino) != "v2" {
		t.Fatalf("atualização: %v %q", err, ler(destino))
	}
	if _, err := os.Stat(novo); !os.IsNotExist(err) {
		t.Fatal("sobrou o .novo")
	}
	// Sem o novo, nada muda e devolve erro.
	if err := trocarArquivo(novo, destino); err == nil || ler(destino) != "v2" {
		t.Fatalf("sem o novo: %v %q", err, ler(destino))
	}
}
