//go:build !windows

package theme

import (
	"os"
	"syscall"
)

// mkfifo crea un FIFO (los tests de archivos que no son regulares).
func mkfifo(path string) error { return syscall.Mkfifo(path, 0o644) }

// releaseFIFO abre el FIFO para escribir sin bloquear: suelta a quien esté esperando para leerlo, y si no hay
// nadie falla en el acto.
func releaseFIFO(path string) {
	if f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
		f.Close()
	}
}
