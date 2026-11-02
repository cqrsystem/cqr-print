//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

const nomeTarefa = "CQR Print"

func pastaPrograma() string {
	base := os.Getenv("ProgramFiles")
	if base == "" {
		base = `C:\Program Files`
	}
	return filepath.Join(base, "CQR Print")
}

// copiarPrograma põe o cqrprint.exe em Arquivos de Programas e devolve o caminho.
func copiarPrograma() (string, error) {
	atual, err := os.Executable()
	if err != nil {
		return "", err
	}
	destino := filepath.Join(pastaPrograma(), "cqrprint.exe")
	if strings.EqualFold(filepath.Clean(atual), filepath.Clean(destino)) {
		return destino, nil
	}
	if err := os.MkdirAll(pastaPrograma(), 0o755); err != nil {
		return "", errors.New("sem permissão: feche e abra de novo com o botão direito → Executar como administrador")
	}
	// Atualização: para a tarefa antes de trocar o arquivo.
	_ = exec.Command("schtasks", "/End", "/TN", nomeTarefa).Run()
	time.Sleep(time.Second)
	origem, err := os.Open(atual)
	if err != nil {
		return "", err
	}
	defer origem.Close()
	tmp := destino + ".novo"
	saida, err := os.Create(tmp)
	if err != nil {
		return "", errors.New("sem permissão para gravar o programa: execute como administrador")
	}
	if _, err := io.Copy(saida, origem); err != nil {
		saida.Close()
		return "", err
	}
	saida.Close()
	if err := trocarArquivo(tmp, destino); err != nil {
		return "", fmt.Errorf("não trocou o programa (feche o CQR Print e tente de novo): %w", err)
	}
	return destino, nil
}

// tarefaXML: roda como SYSTEM ao ligar o computador e confere a cada 5 minutos (se tiver parado,
// volta; se já estiver rodando, não abre outro). Reinicia sozinho em caso de falha.
func tarefaXML(exe string) string {
	return `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.4" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo><Description>CQR Print: imprime as etiquetas do CQR System.</Description></RegistrationInfo>
  <Triggers>
    <BootTrigger><Enabled>true</Enabled></BootTrigger>
    <TimeTrigger>
      <StartBoundary>2026-01-01T00:00:00</StartBoundary>
      <Enabled>true</Enabled>
      <Repetition><Interval>PT5M</Interval><StopAtDurationEnd>false</StopAtDurationEnd></Repetition>
    </TimeTrigger>
  </Triggers>
  <Principals><Principal id="Sistema"><UserId>S-1-5-18</UserId><RunLevel>HighestAvailable</RunLevel></Principal></Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>
    <RestartOnFailure><Interval>PT1M</Interval><Count>999</Count></RestartOnFailure>
    <StartWhenAvailable>true</StartWhenAvailable>
    <Enabled>true</Enabled>
  </Settings>
  <Actions Context="Sistema"><Exec><Command>` + escaparXML(exe) + `</Command><Arguments>executar</Arguments></Exec></Actions>
</Task>`
}

func escaparXML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// registrarTarefa cria (ou troca) a tarefa agendada e já a inicia.
func registrarTarefa(exe string) error {
	arquivo := filepath.Join(os.TempDir(), "cqrprint-tarefa.xml")
	// O schtasks lê o XML em UTF-16 com BOM.
	u := utf16.Encode([]rune(tarefaXML(exe)))
	b := make([]byte, 2+len(u)*2)
	b[0], b[1] = 0xFF, 0xFE
	for i, c := range u {
		b[2+i*2], b[3+i*2] = byte(c), byte(c>>8)
	}
	if err := os.WriteFile(arquivo, b, 0o600); err != nil {
		return err
	}
	defer os.Remove(arquivo)
	if out, err := exec.Command("schtasks", "/Create", "/TN", nomeTarefa, "/XML", arquivo, "/F").CombinedOutput(); err != nil {
		return fmt.Errorf("não criou a tarefa agendada: %s", strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("schtasks", "/Run", "/TN", nomeTarefa).CombinedOutput(); err != nil {
		return fmt.Errorf("não iniciou a tarefa: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func removerTarefa() error {
	_ = exec.Command("schtasks", "/End", "/TN", nomeTarefa).Run()
	if out, err := exec.Command("schtasks", "/Delete", "/TN", nomeTarefa, "/F").CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
