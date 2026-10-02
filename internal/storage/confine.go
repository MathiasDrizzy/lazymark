package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrOutsideNotes es el error de una ruta que no es una nota dentro de la carpeta de notas.
var ErrOutsideNotes = errors.New("la ruta no es una nota de la carpeta de notas")

// ResolveNote valida una ruta que viene de fuera (la CLI, un agente por MCP) antes de leerla o
// escribirla: debe ser un archivo regular .md que, una vez resueltos los enlaces simbólicos y los "..",
// esté dentro de la carpeta de notas. Acepta rutas absolutas o relativas a esa carpeta y devuelve la
// ruta canónica. Todo lo demás da ErrOutsideNotes (o el error del sistema si no existe).
func (s *Storage) ResolveNote(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%w: ruta vacía", ErrOutsideNotes)
	}
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", fmt.Errorf("%w: %q no termina en .md", ErrOutsideNotes, path)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.BaseDir, path)
	}
	base, err := filepath.EvalSymlinks(s.BaseDir)
	if err != nil {
		return "", err
	}
	if base, err = filepath.Abs(base); err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %q está fuera de %q", ErrOutsideNotes, path, s.BaseDir)
	}
	fi, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%w: %q no es un archivo", ErrOutsideNotes, path)
	}
	return real, nil
}
