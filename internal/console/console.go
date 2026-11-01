// Package console faz as perguntas da instalação na janela.
package console

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var entrada = bufio.NewReader(os.Stdin)

// Perguntar mostra a pergunta e devolve a resposta (ou o padrão, se vazia).
func Perguntar(pergunta, padrao string) string {
	if padrao != "" {
		fmt.Printf("%s [%s]: ", pergunta, padrao)
	} else {
		fmt.Printf("%s: ", pergunta)
	}
	linha, _ := entrada.ReadString('\n')
	linha = strings.TrimSpace(linha)
	if linha == "" {
		return padrao
	}
	return linha
}

// Senha pergunta sem mostrar o que é digitado (no Windows).
func Senha(pergunta string) string {
	fmt.Printf("%s: ", pergunta)
	desligarEco()
	linha, _ := entrada.ReadString('\n')
	ligarEco()
	fmt.Println()
	return strings.TrimRight(linha, "\r\n")
}

// Pausar espera o Enter (para a janela não fechar sozinha ao abrir com dois cliques).
func Pausar() {
	fmt.Print("\nPressione Enter para fechar.")
	_, _ = entrada.ReadString('\n')
}
