package storage

import (
	"errors"
	"fmt"
	"io/fs"
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
		// una ruta que no existe y que, además, cae fuera de la carpeta de notas es una ruta inválida (código 2),
		// no una nota que falta: así no se revela qué existe fuera
		if errors.Is(err, os.ErrNotExist) && s.lexicallyOutside(filepath.Clean(path), base) {
			return "", fmt.Errorf("%w: %q está fuera de %q", ErrOutsideNotes, path, s.BaseDir)
		}
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

// lexicallyOutside indica si path, sin resolver enlaces, queda fuera de la carpeta de notas tanto con su ruta tal
// cual como con la ya resuelta (base).
func (s *Storage) lexicallyOutside(path, base string) bool {
	outside := func(root string) bool {
		rel, err := filepath.Rel(root, path)
		return err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel)
	}
	abs, err := filepath.Abs(s.BaseDir)
	return outside(base) && (err != nil || outside(abs))
}

// linkStaysInside indica si la entrada se puede leer como nota: un archivo normal sí; un enlace simbólico solo si
// su destino es una nota de la carpeta de notas (la misma regla que ResolveNote). Así un `enlace.md` hacia un
// archivo de fuera no expone su contenido ni sus tareas.
func (s *Storage) linkStaysInside(path string, d fs.DirEntry) bool {
	if d.Type()&fs.ModeSymlink == 0 {
		return true
	}
	_, err := s.ResolveNote(path)
	return err == nil
}

// confineNewPath valida un destino que puede no existir todavía (restaurar de la papelera): una vez resueltos ".." y
// los enlaces simbólicos del tramo que ya existe, debe quedar dentro de la carpeta de notas.
func (s *Storage) confineNewPath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: la ruta %q no es absoluta", ErrOutsideNotes, path)
	}
	base, err := filepath.EvalSymlinks(s.BaseDir)
	if err != nil {
		return err
	}
	if base, err = filepath.Abs(base); err != nil {
		return err
	}
	existing, rest := filepath.Clean(path), ""
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return fmt.Errorf("%w: %q", ErrOutsideNotes, path)
		}
		rest = filepath.Join(filepath.Base(existing), rest)
		existing = parent
	}
	real, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(base, filepath.Join(real, rest))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: %q está fuera de %q", ErrOutsideNotes, path, s.BaseDir)
	}
	return nil
}

// NotePaths devuelve las rutas de todas las notas .md de la carpeta de notas, sin leerlas: las mismas que ListNotes (sin carpetas
// ocultas ni assets/) y con la misma regla de enlaces simbólicos (uno que sale de la carpeta no cuenta). Las usan la búsqueda y
// todo lo que necesite recorrer las notas sin cargar su contenido.
func (s *Storage) NotePaths() []string {
	var paths []string
	filepath.WalkDir(s.BaseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if (strings.HasPrefix(name, ".") && name != ".") || strings.EqualFold(name, "assets") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") && s.linkStaysInside(path, d) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}
