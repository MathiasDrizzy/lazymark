//go:build windows

package image

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// openRegular abre path solo si es un archivo regular de hasta MaxFileBytes. En Windows no hay O_NONBLOCK: se descartan antes los nombres de
// dispositivo reservados (CON, NUL, COM1…) y lo que Stat dice que no es regular, y se vuelve a comprobar con el archivo ya abierto.
func openRegular(path string) (*os.File, error) {
	base := strings.ToUpper(filepath.Base(path))
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return nil, fmt.Errorf("%s es un nombre de dispositivo de Windows", path)
	}
	if fi, err := os.Stat(path); err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		return nil, fmt.Errorf("%s no es un archivo regular de tamaño admitido", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		_ = f.Close()
		return nil, fmt.Errorf("%s no es un archivo regular de tamaño admitido", path)
	}
	return f, nil
}
