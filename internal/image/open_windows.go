//go:build windows

package image

import (
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"

	"github.com/MathiasDrizzy/lazymark/internal/safeio"
)

// openRegular abre path solo si es un archivo regular de hasta MaxFileBytes. En Windows no hay O_NONBLOCK: se descartan antes los nombres de
// dispositivo reservados (CON, NUL, COM1…) y lo que Stat dice que no es regular, y se vuelve a comprobar con el archivo ya abierto.
func openRegular(path string) (*os.File, error) {
	if safeio.ReservedWindowsName(path) {
		return nil, i18n.Errorf("%s es un nombre de dispositivo de Windows", "%s is a Windows device name", path)
	}
	if fi, err := os.Stat(path); err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		return nil, i18n.Errorf("%s no es un archivo regular de tamaño admitido", "%s is not a regular file of supported size", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		_ = f.Close()
		return nil, i18n.Errorf("%s no es un archivo regular de tamaño admitido", "%s is not a regular file of supported size", path)
	}
	return f, nil
}
