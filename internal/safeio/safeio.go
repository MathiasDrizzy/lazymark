// Package safeio lee archivos que vienen de una carpeta no confiable sin colgarse ni gastar memoria de más: solo archivos regulares (un FIFO o un
// dispositivo como /dev/tty no se abren) y con un tope de tamaño.
package safeio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// WriteFileAtomic escribe data en path sin dejarlo nunca a medio escribir: lo escribe en un temporal de la misma carpeta (nombre aleatorio, O_EXCL, así
// un enlace simbólico preparado de antemano no lo desvía), lo sincroniza y lo renombra encima. Conserva los permisos del archivo que reemplaza
// (o perm si es nuevo). Si algo falla, el archivo original queda como estaba y el temporal se borra.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return WriteFileAtomicIf(path, data, perm, nil)
}

// WriteFileAtomicIf es WriteFileAtomic con un chequeo que corre justo antes del renombrado (ya con el temporal escrito y sincronizado): si devuelve un
// error, no se renombra, el original queda como estaba y se devuelve ese mismo error. Sirve para no pisar un cambio hecho por otro programa mientras
// se escribía. Si path es un enlace simbólico se escribe el archivo al que apunta (el enlace se conserva).
func WriteFileAtomicIf(path string, data []byte, perm os.FileMode, check func() error) error {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		real, err := filepath.EvalSymlinks(path)
		if err != nil { // un enlace roto: escribir a ciegas lo reemplazaría por un archivo; se rechaza
			return fmt.Errorf("%s es un enlace simbólico roto: no se escribe", filepath.Base(path))
		}
		path = real
	}
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".lazymark-*.tmp")
	if err != nil {
		return fmt.Errorf("error al crear archivo temporal: %w", err)
	}
	name := tmp.Name()
	_, werr := tmp.Write(data)
	if werr == nil {
		werr = tmp.Sync()
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr == nil {
		werr = os.Chmod(name, perm)
	}
	if werr == nil && check != nil {
		if cerr := check(); cerr != nil {
			_ = os.Remove(name)
			return cerr
		}
	}
	if werr == nil {
		werr = os.Rename(name, path)
	}
	if werr != nil {
		_ = os.Remove(name)
		return fmt.Errorf("error al escribir %s: %w", filepath.Base(path), werr)
	}
	return nil
}

// reservedWindowsName dice si el último tramo de path es un nombre reservado de Windows (CON, PRN, AUX, NUL, COM1-9, LPT1-9, con o sin extensión,
// sin distinguir mayúsculas): en Windows son dispositivos y abrirlos puede quedarse esperando.
func reservedWindowsName(path string) bool {
	base := path
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	switch b := strings.ToUpper(strings.TrimRight(base, " ")); b {
	case "CON", "PRN", "AUX", "NUL":
		return true
	default:
		if len([]rune(b)) == 4 && (strings.HasPrefix(b, "COM") || strings.HasPrefix(b, "LPT")) {
			switch r := []rune(b)[3]; {
			case r >= '1' && r <= '9', r == '¹', r == '²', r == '³':
				return true
			}
		}
		return false
	}
}
