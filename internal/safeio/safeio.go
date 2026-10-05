// Package safeio lee archivos que vienen de una carpeta no confiable sin colgarse ni gastar memoria de más: solo archivos regulares (un FIFO o un
// dispositivo como /dev/tty no se abren) y con un tope de tamaño.
package safeio

import (
	"errors"
	"fmt"
	"io"
)

var (
	// ErrNotRegular: el archivo no es regular (FIFO, dispositivo, socket, carpeta).
	ErrNotRegular = errors.New("no es un archivo regular")
	// ErrTooLarge: el archivo pesa más que el tope.
	ErrTooLarge = errors.New("el archivo es demasiado grande")
)

// ReadRegular lee path entero si es un archivo regular de como mucho max bytes (max <= 0: sin tope). La comprobación se hace sobre el archivo ya
// abierto (sin bloquear), y la lectura se corta en max+1 bytes por si creció después.
func ReadRegular(path string, max int64) ([]byte, error) {
	f, err := openNonBlocking(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s", ErrNotRegular, path)
	}
	if max > 0 && fi.Size() > max {
		return nil, fmt.Errorf("%w (%d MB, máximo %d MB): %s", ErrTooLarge, fi.Size()>>20, max>>20, path)
	}
	var r io.Reader = f
	if max > 0 {
		r = io.LimitReader(f, max+1)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if max > 0 && int64(len(data)) > max {
		return nil, fmt.Errorf("%w: %s", ErrTooLarge, path)
	}
	return data, nil
}
