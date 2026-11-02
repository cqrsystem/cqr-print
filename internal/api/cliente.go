// Package api fala com o servidor do CQR System (docs/api.md do cqr-system-web, seções 3, 4 e 4.2).
//
// Formato das respostas: {"dados": ..., "meta": {...}} no sucesso e
// {"erro": {"codigo", "mensagem"}} no erro.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Erro devolvido pelo servidor (código e mensagem da API).
type Erro struct {
	Status   int
	Codigo   string
	Mensagem string
}

func (e *Erro) Error() string {
	if e.Mensagem != "" {
		return fmt.Sprintf("%s (%s, HTTP %d)", e.Mensagem, e.Codigo, e.Status)
	}
	return fmt.Sprintf("erro HTTP %d", e.Status)
}

// SemAcesso diz se o servidor recusou o aparelho (token inválido, revogado ou de outro tipo).
// Nesse caso o agente precisa ser ativado de novo.
func SemAcesso(err error) bool {
	var e *Erro
	return errors.As(err, &e) && (e.Status == http.StatusUnauthorized || e.Codigo == "tipo_de_aparelho")
}

// Cliente da API. Token vazio só serve para ativar.
type Cliente struct {
	Servidor string // ex.: https://cqrweb.exemplo.com.br (sem / no fim)
	Token    string
	Versao   string
	HTTP     *http.Client
}

// Novo cria o cliente com tempo limite curto (o agente pergunta de novo em poucos segundos).
func Novo(servidor, token, versao string) *Cliente {
	return &Cliente{
		Servidor: strings.TrimRight(strings.TrimSpace(servidor), "/"),
		Token:    token,
		Versao:   versao,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}
}

// ValidarServidor confere o endereço digitado na instalação: só https (http só para localhost).
func ValidarServidor(servidor string) error {
	u, err := url.Parse(strings.TrimSpace(servidor))
	if err != nil || u.Host == "" {
		return errors.New("endereço inválido; use algo como https://seu-servidor.com.br")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return errors.New("o endereço precisa começar com https://")
	}
	return nil
}

func (c *Cliente) pedir(ctx context.Context, metodo, caminho string, corpo any, saida any) error {
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, c.Servidor+"/api/v1/"+strings.TrimLeft(caminho, "/"), leitor)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CQRPrint/"+c.Versao)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Etiquetas em imagem podem ser grandes, mas não passam de alguns MB por pedido.
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var env struct {
			Erro struct {
				Codigo   string `json:"codigo"`
				Mensagem string `json:"mensagem"`
			} `json:"erro"`
		}
		_ = json.Unmarshal(dados, &env)
		return &Erro{Status: resp.StatusCode, Codigo: env.Erro.Codigo, Mensagem: env.Erro.Mensagem}
	}
	if saida == nil {
		return nil
	}
	env := struct {
		Dados json.RawMessage `json:"dados"`
	}{}
	if err := json.Unmarshal(dados, &env); err != nil {
		return fmt.Errorf("resposta inesperada do servidor: %w", err)
	}
	return json.Unmarshal(env.Dados, saida)
}

// Ativacao: resposta de POST /aparelhos/ativar.
type Ativacao struct {
	Token    string `json:"token"`
	Aparelho struct {
		ID   int    `json:"id"`
		Nome string `json:"nome"`
	} `json:"aparelho"`
	Empresa struct {
		Nome string `json:"nome"`
	} `json:"empresa"`
}

// Ativar registra este computador como agente de impressão da empresa (CNPJ, matrícula e senha de
// Supervisor ou Administrador). O token volta uma vez só.
func (c *Cliente) Ativar(ctx context.Context, cnpj, matricula, senha, nome string) (*Ativacao, error) {
	var a Ativacao
	err := c.pedir(ctx, http.MethodPost, "aparelhos/ativar", map[string]string{
		"cnpj": cnpj, "matricula": matricula, "senha": senha, "tipo": "impressao", "nome": nome,
	}, &a)
	if err != nil {
		return nil, err
	}
	if a.Token == "" {
		return nil, errors.New("o servidor não devolveu o token")
	}
	return &a, nil
}

// Impressora configurada no Desk para este agente.
type Impressora struct {
	ID          int    `json:"id"`
	Codigo      string `json:"codigo"`
	Destino     string `json:"destino"` // windows ou rede
	NomeWindows string `json:"nome_windows"`
	Endereco    string `json:"endereco"`
	Porta       int    `json:"porta"`
	Linguagem   string `json:"linguagem"` // zpl ou imagem
	DPI         int    `json:"dpi"`
}

// ImpressoraDoWindows: o que o agente enxerga no computador.
type ImpressoraDoWindows struct {
	Nome   string `json:"nome"`
	Padrao bool   `json:"padrao"`
}

// Sinal: resposta de POST /impressao/sinal.
type Sinal struct {
	Empresa      string       `json:"empresa"`
	IntervaloSeg int          `json:"intervalo_seg"`
	Impressoras  []Impressora `json:"impressoras"`
	Agente       struct {
		ID   int    `json:"id"`
		Nome string `json:"nome"`
	} `json:"agente"`
}

// EnviarSinal informa as impressoras do Windows e recebe as configuradas.
func (c *Cliente) EnviarSinal(ctx context.Context, versao string, impressoras []ImpressoraDoWindows) (*Sinal, error) {
	if impressoras == nil {
		impressoras = []ImpressoraDoWindows{}
	}
	var s Sinal
	err := c.pedir(ctx, http.MethodPost, "impressao/sinal", map[string]any{"versao": versao, "impressoras": impressoras}, &s)
	return &s, err
}

// Trabalho de impressão (GET /impressao/trabalhos).
type Trabalho struct {
	ID         int        `json:"id"`
	Impressora Impressora `json:"impressora"`
	Descricao  string     `json:"descricao"`
	Linguagem  string     `json:"linguagem"`
	Conteudo   string     `json:"conteudo"`
	Formato    string     `json:"formato"` // texto (ZPL) ou png_base64
	Copias     int        `json:"copias"`
	LarguraMM  float64    `json:"largura_mm"`
	AlturaMM   float64    `json:"altura_mm"`
	DPI        int        `json:"dpi"`
}

// BuscarTrabalhos pega os trabalhos pendentes (o servidor marca como enviados).
func (c *Cliente) BuscarTrabalhos(ctx context.Context, limite int) ([]Trabalho, error) {
	var t []Trabalho
	err := c.pedir(ctx, http.MethodGet, "impressao/trabalhos?limite="+strconv.Itoa(limite), nil, &t)
	return t, err
}

// Resultado de um trabalho.
type Resultado struct {
	ID   int    `json:"id"`
	OK   bool   `json:"ok"`
	Erro string `json:"erro,omitempty"`
}

// EnviarResultados responde vários trabalhos de uma vez. Devolve os ids aceitos.
func (c *Cliente) EnviarResultados(ctx context.Context, resultados []Resultado) ([]int, error) {
	var r struct {
		Aceitos []int `json:"aceitos"`
	}
	err := c.pedir(ctx, http.MethodPost, "impressao/resultados", map[string]any{"resultados": resultados}, &r)
	return r.Aceitos, err
}
