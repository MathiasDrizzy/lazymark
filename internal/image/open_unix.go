//go:build !windows

package image

import (
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
	"syscall"
)

// openRegular abre path solo si es un archivo regular de hasta MaxFileBytes. Se abre sin bloquear (O_NONBLOCK): abrir un FIFO o /dev/tty con
// os.Open espera para siempre; aquí se abre al instante, se comprueba con fstat y se cierra sin leer nada.
func openRegular(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	if err := checkRegular(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

func checkRegular(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return i18n.Errorf("%s no es un archivo regular", "%s is not a regular file", f.Name())
	}
	if fi.Size() > MaxFileBytes {
		return i18n.Errorf("%s pesa más de %d MB", "%s is larger than %d MB", f.Name(), MaxFileBytes>>20)
	}
	return nil
}
