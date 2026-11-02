//go:build windows

package config

import (
	"encoding/base64"
	"syscall"
	"unsafe"
)

var (
	crypt32              = syscall.NewLazyDLL("crypt32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtectData = crypt32.NewProc("CryptProtectData")
	procCryptUnprotect   = crypt32.NewProc("CryptUnprotectData")
	procLocalFree        = kernel32.NewProc("LocalFree")
)

type blob struct {
	n uint32
	p *byte
}

// CRYPTPROTECT_LOCAL_MACHINE: o agente roda como SYSTEM (tarefa agendada) e quem instala é o
// administrador; os dois precisam abrir o token, que só vale neste computador.
const protegerMaquina = 0x4
const semInterface = 0x1

func novoBlob(b []byte) *blob {
	if len(b) == 0 {
		return &blob{}
	}
	return &blob{n: uint32(len(b)), p: &b[0]}
}

func (b *blob) bytes() []byte {
	out := make([]byte, b.n)
	copy(out, unsafe.Slice(b.p, b.n))
	return out
}

func proteger(texto string) (string, error) {
	var saida blob
	r, _, err := procCryptProtectData.Call(uintptr(unsafe.Pointer(novoBlob([]byte(texto)))), 0, 0, 0, 0,
		protegerMaquina|semInterface, uintptr(unsafe.Pointer(&saida)))
	if r == 0 {
		return "", err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(saida.p)))
	return base64.StdEncoding.EncodeToString(saida.bytes()), nil
}

func abrir(protegido string) (string, error) {
	dados, err := base64.StdEncoding.DecodeString(protegido)
	if err != nil {
		return "", err
	}
	var saida blob
	r, _, err := procCryptUnprotect.Call(uintptr(unsafe.Pointer(novoBlob(dados))), 0, 0, 0, 0, semInterface, uintptr(unsafe.Pointer(&saida)))
	if r == 0 {
		return "", err
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(saida.p)))
	return string(saida.bytes()), nil
}
