//go:build windows

package safeio

import (
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
)

// openNonBlocking abre path en Windows, donde no hay O_NONBLOCK: antes de abrir se descartan los nombres de dispositivo reservados (CON, NUL, COM1…) y
// lo que Stat dice que no es un archivo regular (una tubería con nombre, un dispositivo), para no quedarse esperando en la apertura o la lectura.
func openNonBlocking(path string) (*os.File, error) {
	if ReservedWindowsName(path) {
		return nil, i18n.Errorf("%w: %s es un nombre de dispositivo de Windows", "%w: %s is a Windows device name", ErrNotRegular, path)
	}
	if fi, err := os.Stat(path); err == nil && !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s", ErrNotRegular, path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	// vuelve a comprobar con el archivo ya abierto: entre el Stat y el Open pudo cambiar (TOCTOU)
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("%w: %s", ErrNotRegular, path)
	}
	return f, nil
}
