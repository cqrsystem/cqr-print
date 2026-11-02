package main

import "os"

// trocarArquivo põe o novo no lugar do antigo. O Windows não deixa apagar nem sobrescrever um
// programa aberto, mas deixa renomeá-lo: o antigo vira .antigo (apagado na próxima vez que der).
func trocarArquivo(novo, destino string) error {
	antigo := destino + ".antigo"
	_ = os.Remove(antigo)
	if err := os.Rename(novo, destino); err == nil {
		return nil
	}
	if err := os.Rename(destino, antigo); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(novo, destino); err != nil {
		_ = os.Rename(antigo, destino)
		return err
	}
	return nil
}

// limparAntigo apaga a cópia anterior deixada por uma atualização (quando já não está aberta).
func limparAntigo() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".antigo")
	}
}
