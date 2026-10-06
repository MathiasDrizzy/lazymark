package storage

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrTaskNotFound es el error de un id de tarea que no corresponde a ninguna tarea.
var ErrTaskNotFound = i18n.NewError("no existe esa tarea", "no such task")

// taskHash es la huella de una tarea: el texto sin etiquetas del tablero, en minúscula y con los espacios
// colapsados. No depende de la casilla ni de la columna (mover o marcar no cambia la identidad) ni de la línea
// (editar o insertar otras líneas tampoco).
func taskHash(t Task) string {
	norm := strings.ToLower(strings.Join(strings.Fields(CleanTaskText(t.Text)), " "))
	sum := sha1.Sum([]byte(norm))
	return hex.EncodeToString(sum[:])[:8]
}

// relPath es la ruta de la nota relativa a la carpeta de notas, con "/".
func (s *Storage) relPath(path string) string {
	rel, err := filepath.Rel(s.BaseDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// TaskIDs devuelve el id de cada tarea de la nota, en el mismo orden que n.Tasks:
// `<ruta relativa>#<huella de 8 hex>`, y `.<n>` desde la segunda tarea con el mismo texto de la misma nota. Una
// tarea conserva su id al editar o mover otras líneas, al cambiar de columna o al marcarla; cambia si se edita
// su propio texto.
func (s *Storage) TaskIDs(n Note) []string {
	rel := s.relPath(n.Path)
	seen := map[string]int{}
	ids := make([]string, len(n.Tasks))
	for i, t := range n.Tasks {
		h := taskHash(t)
		seen[h]++
		ids[i] = rel + "#" + h
		if seen[h] > 1 {
			ids[i] += "." + strconv.Itoa(seen[h])
		}
	}
	return ids
}

// FindTask busca la tarea con ese id en las notas de la carpeta.
func (s *Storage) FindTask(id string) (Note, Task, error) {
	i := strings.LastIndex(id, "#")
	if i <= 0 || i == len(id)-1 {
		return Note{}, Task{}, i18n.Errorf("%w: el id %q no es <nota.md>#<huella>", "%w: id %q is not <note.md>#<hash>", ErrTaskNotFound, id)
	}
	rel := id[:i]
	notes, err := s.ListNotes()
	if err != nil {
		return Note{}, Task{}, err
	}
	for _, n := range notes {
		if s.relPath(n.Path) != rel {
			continue
		}
		for j, tid := range s.TaskIDs(n) {
			if tid == id {
				return n, n.Tasks[j], nil
			}
		}
	}
	return Note{}, Task{}, fmt.Errorf("%w: %q", ErrTaskNotFound, id)
}

// ResolveFolder valida una subcarpeta (relativa a la carpeta de notas; "" es la carpeta de notas misma): debe
// existir y, resueltos "..", y symlinks, quedar dentro de la carpeta de notas.
func (s *Storage) ResolveFolder(rel string) (string, error) {
	path := s.BaseDir
	if rel != "" {
		if filepath.IsAbs(rel) {
			return "", i18n.Errorf("%w: la carpeta debe ser relativa a la carpeta de notas", "%w: folder must be relative to the notes folder", ErrOutsideNotes)
		}
		path = filepath.Join(s.BaseDir, rel)
	}
	base, err := filepath.EvalSymlinks(s.BaseDir)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", i18n.Errorf("%w: la carpeta %q no existe", "%w: folder %q does not exist", ErrOutsideNotes, rel)
		}
		return "", err
	}
	r, err := filepath.Rel(base, real)
	if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", i18n.Errorf("%w: %q está fuera de la carpeta de notas", "%w: %q is outside the notes folder", ErrOutsideNotes, rel)
	}
	if fi, err := os.Stat(real); err != nil || !fi.IsDir() {
		return "", i18n.Errorf("%w: %q no es una carpeta", "%w: %q is not a folder", ErrOutsideNotes, rel)
	}
	return real, nil
}

// EnsureFolder devuelve la carpeta rel (relativa a la carpeta de notas, con "/") y la crea, con las que falten en el camino, si no
// existe. Solo crea dentro de la carpeta de notas: rechaza rutas absolutas, ".." y nombres vacíos o con caracteres de control, y no
// sigue enlaces simbólicos que salgan de ella (ResolveFolder lo comprueba en cada tramo).
func (s *Storage) EnsureFolder(rel string) (string, error) {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." {
		return s.BaseDir, nil
	}
	cur := ""
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.ContainsAny(seg, "\\:") || strings.IndexFunc(seg, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
			return "", i18n.Errorf("%w: la carpeta %q no es válida", "%w: folder %q is not valid", ErrOutsideNotes, rel)
		}
		cur = path.Join(cur, seg)
		if _, err := os.Lstat(filepath.Join(s.BaseDir, filepath.FromSlash(cur))); errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(filepath.Join(s.BaseDir, filepath.FromSlash(cur)), 0o755); err != nil {
				return "", err
			}
		}
		if _, err := s.ResolveFolder(filepath.FromSlash(cur)); err != nil {
			return "", err
		}
	}
	// la ruta se devuelve bajo la carpeta de notas tal como la ven las demás (no la real: en macOS /var es /private/var)
	return filepath.Join(s.BaseDir, filepath.FromSlash(cur)), nil
}
