//go:build !windows

package main

import "errors"

var errSoWindows = errors.New("a instalação é só no Windows")

func pastaPrograma() string           { return "" }
func copiarPrograma() (string, error) { return "", errSoWindows }
func registrarTarefa(string) error    { return errSoWindows }
func removerTarefa() error            { return errSoWindows }
