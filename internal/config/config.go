// Package config guarda a configuração do agente: endereço do servidor, nome e o token do
// aparelho. No Windows o token fica protegido pelo DPAPI da máquina (só este computador abre).
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config do agente.
type Config struct {
	Servidor string `json:"servidor"`
	Nome     string `json:"nome"`
	Empresa  string `json:"empresa"`
	Token    string `json:"-"`
	// TokenProtegido: o token cifrado (DPAPI no Windows), em base64.
	TokenProtegido string `json:"token_protegido"`
}

// Pasta dos dados do agente (ProgramData no Windows).
func Pasta() string {
	if p := os.Getenv("CQRPRINT_PASTA"); p != "" {
		return p
	}
	if d := os.Getenv("ProgramData"); d != "" {
		return filepath.Join(d, "CQR Print")
	}
	return filepath.Join(os.TempDir(), "cqr-print")
}

func arquivo() string { return filepath.Join(Pasta(), "config.json") }

// ErrSemConfig: o agente ainda não foi ativado.
var ErrSemConfig = errors.New("agente ainda não ativado")

// Carregar lê a configuração e abre o token.
func Carregar() (*Config, error) {
	b, err := os.ReadFile(arquivo())
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrSemConfig
	}
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.TokenProtegido == "" || c.Servidor == "" {
		return nil, ErrSemConfig
	}
	token, err := abrir(c.TokenProtegido)
	if err != nil {
		return nil, errors.New("não foi possível abrir o token (o arquivo é de outro computador?); ative de novo")
	}
	c.Token = token
	return &c, nil
}

// Salvar grava a configuração com o token protegido.
func (c *Config) Salvar() error {
	protegido, err := proteger(c.Token)
	if err != nil {
		return err
	}
	c.TokenProtegido = protegido
	if err := os.MkdirAll(Pasta(), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := arquivo() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, arquivo())
}

// Apagar remove a configuração (desinstalar ou ativar de novo).
func Apagar() error {
	err := os.Remove(arquivo())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
