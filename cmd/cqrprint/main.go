// CQR Print: agente de impressão de etiquetas do CQR System para Windows.
//
// Dois cliques no cqrprint.exe (como administrador) instalam: pergunta o endereço do servidor, o
// CNPJ, a matrícula e a senha de um Supervisor ou Administrador, ativa este computador como
// "Agente de impressão", copia o programa para Arquivos de Programas e cria a tarefa agendada que
// roda como SYSTEM ao ligar o computador.
//
// Comandos: instalar, executar, ativar, impressoras, testar "<impressora>", status, desinstalar, versao.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/cqrsystem/cqr-print/internal/agente"
	"github.com/cqrsystem/cqr-print/internal/api"
	"github.com/cqrsystem/cqr-print/internal/config"
	"github.com/cqrsystem/cqr-print/internal/console"
	"github.com/cqrsystem/cqr-print/internal/impressao"
	"github.com/cqrsystem/cqr-print/internal/registro"
)

// Definidos na compilação (-ldflags "-X main.versao=1.0.0 -X main.servidorPadrao=https://...").
// O endereço não fica no código: cada instalação (nuvem ou on-premise) tem o seu.
var (
	versao         = "dev"
	servidorPadrao = ""
)

func main() {
	comando := ""
	if len(os.Args) > 1 {
		comando = strings.ToLower(os.Args[1])
	}
	var err error
	switch comando {
	case "", "instalar":
		err = instalar(comando == "")
	case "executar":
		err = executar()
	case "ativar":
		_, err = ativar()
	case "impressoras":
		err = listarImpressoras()
	case "testar":
		err = testar(os.Args[2:])
	case "status":
		err = status()
	case "desinstalar":
		err = desinstalar()
	case "versao", "--version", "-v":
		fmt.Println("CQR Print", versao)
	default:
		fmt.Println("Uso: cqrprint [instalar | executar | ativar | impressoras | testar \"<impressora>\" | status | desinstalar | versao]")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		if comando == "" {
			console.Pausar()
		}
		os.Exit(1)
	}
}

func instalar(doisCliques bool) error {
	fmt.Println("CQR Print", versao, "— instalação")
	fmt.Println()
	exe, err := copiarPrograma()
	if err != nil {
		return err
	}
	cfg, err := config.Carregar()
	if err != nil {
		if cfg, err = ativar(); err != nil {
			return err
		}
	} else {
		fmt.Printf("Já ativado para %s (%s).\n", cfg.Empresa, cfg.Servidor)
	}
	if err := registrarTarefa(exe); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Pronto! O CQR Print está rodando e volta sozinho quando o computador for ligado.")
	fmt.Println("Agora, no Desk: Configurações → Impressoras → cadastre as impressoras deste computador.")
	fmt.Println("Registro:", config.Pasta())
	if doisCliques {
		console.Pausar()
	}
	return nil
}

func ativar() (*config.Config, error) {
	fmt.Println("Ative este computador com um Supervisor ou Administrador do CQR System.")
	servidor := console.Perguntar("Endereço do CQR System", servidorPadrao)
	if err := api.ValidarServidor(servidor); err != nil {
		return nil, err
	}
	cnpj := console.Perguntar("CNPJ da empresa", "")
	matricula := console.Perguntar("Matrícula", "")
	senha := console.Senha("Senha")
	host, _ := os.Hostname()
	nome := console.Perguntar("Nome deste computador no CQR System", host)

	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	cliente := api.Novo(servidor, "", versao)
	a, err := cliente.Ativar(ctx, cnpj, matricula, senha, nome)
	if err != nil {
		return nil, fmt.Errorf("não ativou: %w", err)
	}
	cfg := &config.Config{Servidor: cliente.Servidor, Nome: a.Aparelho.Nome, Empresa: a.Empresa.Nome, Token: a.Token}
	if err := cfg.Salvar(); err != nil {
		return nil, fmt.Errorf("ativou, mas não gravou a configuração: %w", err)
	}
	fmt.Printf("Ativado: %s, na empresa %s.\n", cfg.Nome, cfg.Empresa)
	return cfg, nil
}

func executar() error {
	registro.Iniciar(config.Pasta(), true)
	cfg, err := config.Carregar()
	if err != nil {
		registro.Linha("não iniciou: %v", err)
		return err
	}
	registro.Linha("CQR Print %s iniciado (%s, %s)", versao, cfg.Nome, cfg.Servidor)
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()
	a := agente.Novo(api.Novo(cfg.Servidor, cfg.Token, versao), impressao.Sistema{}, versao)
	err = a.Rodar(ctx)
	if errors.Is(err, agente.ErrDesativado) {
		registro.Linha("%v", err)
		// Espera antes de sair, para a tarefa agendada não ficar reiniciando em seguida.
		time.Sleep(10 * time.Minute)
		return err
	}
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func listarImpressoras() error {
	lista, err := impressao.Sistema{}.Listar()
	if err != nil {
		return err
	}
	if len(lista) == 0 {
		fmt.Println("Nenhuma impressora encontrada.")
	}
	for _, i := range lista {
		marca := ""
		if i.Padrao {
			marca = " (padrão)"
		}
		fmt.Println("-", i.Nome+marca)
	}
	fmt.Println("\nObs.: o CQR Print roda como SYSTEM: impressora conectada só para um usuário pode não aparecer para ele. Instale a impressora para todos os usuários.")
	return nil
}

// testar manda uma etiqueta ZPL de teste: cqrprint testar "ZDesigner ZD220" ou cqrprint testar --ip 192.168.0.50
func testar(args []string) error {
	if len(args) == 0 {
		return errors.New(`informe a impressora: cqrprint testar "Nome da impressora"  ou  cqrprint testar --ip 192.168.0.50`)
	}
	zpl := "^XA^CI28^FO30,30^A0N,40,40^FDCQR Print^FS^FO30,85^A0N,28,28^FD" + time.Now().Format("02/01/2006 15:04") + "^FS^XZ\n"
	if args[0] == "--ip" && len(args) > 1 {
		return impressao.EnviarRede(args[1], 9100, []byte(zpl))
	}
	err := impressao.Sistema{}.Imprimir(api.Trabalho{Descricao: "Teste", Linguagem: "zpl", Formato: "texto", Conteudo: zpl,
		Impressora: api.Impressora{Destino: "windows", NomeWindows: strings.Join(args, " ")}})
	if err == nil {
		fmt.Println("Enviado. Se a impressora não for Zebra (ZPL), nada sai: cadastre-a como \"Outras\" no Desk.")
	}
	return err
}

func status() error {
	cfg, err := config.Carregar()
	if err != nil {
		return err
	}
	fmt.Printf("CQR Print %s\nEmpresa: %s\nComputador: %s\nServidor: %s\n", versao, cfg.Empresa, cfg.Nome, cfg.Servidor)
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	lista, _ := impressao.Sistema{}.Listar()
	s, err := api.Novo(cfg.Servidor, cfg.Token, versao).EnviarSinal(ctx, versao, lista)
	if err != nil {
		return fmt.Errorf("sem contato com o servidor: %w", err)
	}
	fmt.Printf("Conectado. Impressoras configuradas no Desk: %d\n", len(s.Impressoras))
	for _, i := range s.Impressoras {
		onde := i.NomeWindows
		if i.Destino == "rede" {
			onde = fmt.Sprintf("%s:%d", i.Endereco, i.Porta)
		}
		fmt.Printf("- %s (%s): %s\n", i.Codigo, i.Linguagem, onde)
	}
	return nil
}

func desinstalar() error {
	if err := removerTarefa(); err != nil {
		fmt.Println("Aviso:", err)
	}
	if err := config.Apagar(); err != nil {
		return err
	}
	fmt.Println("CQR Print removido deste computador. Revogue o agente no Desk (Configurações → Aparelhos).")
	fmt.Println("Pode apagar a pasta", pastaPrograma())
	return nil
}
