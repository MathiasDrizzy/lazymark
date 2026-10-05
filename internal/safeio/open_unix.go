//go:build !windows

package safeio

import (
	"os"
	"syscall"
)

// openNonBlocking abre sin esperar (O_NONBLOCK): abrir un FIFO con os.Open se queda esperando para siempre.
func openNonBlocking(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
