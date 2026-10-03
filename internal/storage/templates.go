package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Carpetas especiales dentro de la carpeta de notas.
const (
	TemplatesDir  = "templates" // las plantillas de nota: templates/<nombre>.md
	JournalDir    = "journal"   // las notas diarias: journal/AAAA-MM-DD.md
	DailyTemplate = "daily"     // la plantilla de la nota diaria: templates/daily.md
)

// maxTemplateBytes es el tamaño máximo de una plantilla (una plantilla enorme es un error, no una nota).
const maxTemplateBytes = 256 << 10

// templatePath devuelve la ruta (ya confinada a la carpeta de notas) de la plantilla name, con o sin ".md".
func (s *Storage) templatePath(name string) (string, error) {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".md")
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\:`) || strings.IndexFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return "", fmt.Errorf("%w: nombre de plantilla no válido %q", ErrTemplateNotFound, name)
	}
	p, err := s.ResolveNote(filepath.Join(s.BaseDir, TemplatesDir, name+".md"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
		}
		return "", err
	}
	return p, nil
}

// ErrTemplateNotFound es el error cuando la plantilla pedida no existe en templates/ (o su nombre no vale).
var ErrTemplateNotFound = errors.New("no existe la plantilla")

// Templates lista los nombres (sin ".md") de las plantillas de templates/, en orden alfabético. Sin carpeta, lista vacía.
func (s *Storage) Templates() []string {
	dir, err := s.ResolveFolder(TemplatesDir)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(n), ".md") || strings.HasPrefix(n, ".") {
			continue
		}
		if _, err := s.templatePath(n); err != nil { // un enlace simbólico que sale de la carpeta de notas no es una plantilla
			continue
		}
		names = append(names, strings.TrimSuffix(n, filepath.Ext(n)))
	}
	sort.Strings(names)
	return names
}

// RenderTemplate lee la plantilla name y cambia {{date}} por la fecha (AAAA-MM-DD), {{time}} por la hora (HH:MM) y {{title}} por el
// título. Lo demás, incluidas otras {{llaves}}, queda tal cual; el título no se vuelve a expandir.
func (s *Storage) RenderTemplate(name, title string, now time.Time) (string, error) {
	p, err := s.templatePath(name)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	if fi.Size() > maxTemplateBytes {
		return "", fmt.Errorf("la plantilla %q pesa más de %d KB", name, maxTemplateBytes>>10)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return strings.NewReplacer(
		"{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04"),
		"{{title}}", title,
	).Replace(string(data)), nil
}

// CreateNoteFromTemplate crea la nota title en dir con el contenido de la plantilla template.
func (s *Storage) CreateNoteFromTemplate(dir, title, template string, now time.Time) (*Note, error) {
	body, err := s.RenderTemplate(template, title, now)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		body = "# " + title + "\n" // una plantilla vacía da una nota con su título, no la nota por defecto
	}
	return s.CreateNoteInDirWithBody(dir, title, body)
}

// DailyNote devuelve la nota diaria de now (journal/AAAA-MM-DD.md) y si la acaba de crear. Si no existe la crea con la plantilla
// templates/daily.md (o, sin ella, con el título de la fecha), y crea journal/ si falta. Si ya existe, no la toca.
func (s *Storage) DailyNote(now time.Time) (*Note, bool, error) {
	name := now.Format("2006-01-02")
	dir, err := s.EnsureFolder(JournalDir)
	if err != nil {
		return nil, false, err
	}
	existing := func() (*Note, bool, error) {
		p := filepath.Join(dir, name+".md")
		if _, err := s.ResolveNote(p); err != nil { // confinada: un enlace simbólico hacia fuera no se lee
			return nil, false, err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, false, err
		}
		n := &Note{ID: filepath.Base(p), Title: name, Path: p, Content: string(data)}
		n.Tags, n.Tasks = s.extractTags(n.Content), s.extractTasks(name, p, n.Content)
		return n, false, nil
	}
	if _, err := os.Lstat(filepath.Join(dir, name+".md")); err == nil {
		return existing()
	}
	var note *Note
	if _, terr := s.templatePath(DailyTemplate); terr == nil {
		note, err = s.CreateNoteFromTemplate(dir, name, DailyTemplate, now)
	} else if errors.Is(terr, ErrTemplateNotFound) {
		note, err = s.CreateNoteInDirWithBody(dir, name, "# "+name+"\n\n")
	} else {
		return nil, false, terr
	}
	if errors.Is(err, ErrNoteExists) { // otra instancia la creó entre medio
		return existing()
	}
	if err != nil {
		return nil, false, err
	}
	return note, true, nil
}

// inTemplates dice si path está dentro de templates/ (en la raíz de la carpeta de notas).
func (s *Storage) inTemplates(path string) bool {
	rel, err := filepath.Rel(s.BaseDir, path)
	return err == nil && (rel == TemplatesDir || strings.HasPrefix(filepath.ToSlash(rel), TemplatesDir+"/"))
}
