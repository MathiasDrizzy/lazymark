//go:build windows

package image

import (
	"fmt"
	"os"
)

// openRegular abre path solo si es un archivo regular de hasta MaxFileBytes (en Windows no hay FIFO que bloquee la apertura).
func openRegular(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		_ = f.Close()
		return nil, fmt.Errorf("%s no es un archivo regular de tamaño admitido", path)
	}
	return f, nil
}
