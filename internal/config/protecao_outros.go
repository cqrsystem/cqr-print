//go:build !windows

package config

import "encoding/base64"

// Fora do Windows (só para desenvolvimento e testes) o token fica apenas codificado.
func proteger(texto string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(texto)), nil
}

func abrir(protegido string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(protegido)
	return string(b), err
}
