//go:build !windows

package impressao

import (
	"errors"
	"image"

	"github.com/cqrsystem/cqr-print/internal/api"
)

var errSoWindows = errors.New("impressão pelo Windows: este computador não é Windows")

func listar() ([]api.ImpressoraDoWindows, error) { return nil, nil }

func enviarRaw(_, _ string, _ []byte) error { return errSoWindows }

func imprimirImagem(_, _ string, _ image.Image, _, _ float64, _ int) error { return errSoWindows }
