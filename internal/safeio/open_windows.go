//go:build windows

package safeio

import (
	"fmt"
	"os"
)

// openNonBlocking abre path en Windows, donde no hay O_NONBLOCK: antes de abrir se descartan los nombres de dispositivo reservados (CON, NUL, COM1…) y
// lo que Stat dice que no es un archivo regular (una tubería con nombre, un dispositivo), para no quedarse esperando en la apertura o la lectura.
func openNonBlocking(path string) (*os.File, error) {
	if reservedWindowsName(path) {
		return nil, fmt.Errorf("%w: %s es un nombre de dispositivo de Windows", ErrNotRegular, path)
	}
	if fi, err := os.Stat(path); err == nil && !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s", ErrNotRegular, path)
	}
	return os.Open(path)
}
