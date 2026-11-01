//go:build windows

package console

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
	modoAntes          uint32
)

const ecoEntrada = 0x4 // ENABLE_ECHO_INPUT

func desligarEco() {
	h := os.Stdin.Fd()
	if r, _, _ := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&modoAntes))); r != 0 {
		procSetConsoleMode.Call(h, uintptr(modoAntes&^ecoEntrada))
	}
}

func ligarEco() {
	if modoAntes != 0 {
		procSetConsoleMode.Call(os.Stdin.Fd(), uintptr(modoAntes))
	}
}
