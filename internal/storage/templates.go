package storage

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Carpetas especiales dentro de la carpeta de notas (los valores por defecto: `templates_folder`, `daily_folder` y `daily_name` en la configuración los cambian).
const (
	TemplatesDir  = "templates" // las plantillas de nota: templates/<nombre>.md
	JournalDir    = "journal"   // las notas diarias: journal/AAAA-MM-DD.md
	DailyTemplate = "daily"     // la plantilla de la nota diaria: templates/daily.md
)

// maxTemplateBytes es el tamaño máximo de una plantilla (una plantilla enorme es un error, no una nota).
const maxTemplateBytes = 256 << 10

// ErrTemplateInvalid es el error de una plantilla que no se puede usar: no es texto UTF-8 (bytes nulos, UTF-16, otra codificación) o pesa
// más de maxTemplateBytes. No se crea ninguna nota con ella.
var ErrTemplateInvalid = i18n.NewError("plantilla no válida", "invalid template")

// templateVar reconoce una {{variable}}: el nombre es lo que haya entre las llaves, sin llaves ni saltos de línea.
var templateVar = regexp.MustCompile(`\{\{[^{}\n]{1,40}\}\}`)

// templatePath devuelve la ruta (ya confinada a la carpeta de notas) de la plantilla name, con o sin ".md".
func (s *Storage) templatePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if strings.HasSuffix(strings.ToLower(name), ".md") {
		name = name[:len(name)-3]
	}
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\:`) || strings.IndexFunc(name, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return "", i18n.Errorf("%w: nombre de plantilla no válido %q", "%w: invalid template name %q", ErrTemplateNotFound, name)
	}
	p, err := s.ResolveNote(s.templateFile(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
		}
		return "", err
	}
	return p, nil
}

// ErrTemplateNotFound es el error cuando la plantilla pedida no existe en templates/ (o su nombre no vale).
var ErrTemplateNotFound = i18n.NewError("no existe la plantilla", "template does not exist")

// Templates lista los nombres (sin ".md") de las plantillas de templates/, en orden alfabético. Sin carpeta, lista vacía.
func (s *Storage) Templates() []string {
	dir, err := s.ResolveFolder(filepath.FromSlash(s.templatesRel()))
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
// título; los nombres no distinguen mayúsculas ({{Date}} vale). Lo demás, incluidas las variables desconocidas, queda tal cual; el título
// no se vuelve a expandir. Devuelve ErrTemplateInvalid si la plantilla no es texto UTF-8 o pesa más de 256 KB.
func (s *Storage) RenderTemplate(name, title string, now time.Time) (string, error) {
	out, _, err := s.renderTemplate(name, title, now)
	return out, err
}

// renderTemplate es RenderTemplate y además devuelve las {{variables}} desconocidas que dejó tal cual, sin repetir, en el orden en que
// aparecen.
func (s *Storage) renderTemplate(name, title string, now time.Time) (string, []string, error) {
	p, err := s.templatePath(name)
	if err != nil {
		return "", nil, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return "", nil, err
	}
	if !fi.Mode().IsRegular() { // una tubería o un dispositivo no es una plantilla (y leerlo podría bloquear)
		return "", nil, i18n.Errorf("%w: %q no es un archivo", "%w: %q is not a file", ErrTemplateInvalid, name)
	}
	if fi.Size() > maxTemplateBytes {
		return "", nil, i18n.Errorf("%w: %q pesa más de %d KB", "%w: %q is larger than %d KB", ErrTemplateInvalid, name, maxTemplateBytes>>10)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", nil, err
	}
	if len(data) > maxTemplateBytes { // creció entre el Stat y la lectura
		return "", nil, i18n.Errorf("%w: %q pesa más de %d KB", "%w: %q is larger than %d KB", ErrTemplateInvalid, name, maxTemplateBytes>>10)
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return "", nil, i18n.Errorf("%w: %q no es texto UTF-8", "%w: %q is not UTF-8 text", ErrTemplateInvalid, name)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // el BOM de UTF-8 no pasa a la nota
	var unknown []string
	seen := map[string]bool{}
	out := templateVar.ReplaceAllStringFunc(string(data), func(m string) string {
		switch strings.ToLower(m[2 : len(m)-2]) {
		case "date":
			return now.Format("2006-01-02")
		case "time":
			return now.Format("15:04")
		case "title":
			return title
		}
		if !seen[m] {
			seen[m] = true
			unknown = append(unknown, m)
		}
		return m
	})
	return out, unknown, nil
}

// CreateNoteFromTemplate crea la nota title en dir con el contenido de la plantilla template.
func (s *Storage) CreateNoteFromTemplate(dir, title, template string, now time.Time) (*Note, error) {
	body, unknown, err := s.renderTemplate(template, title, now)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		body = "# " + title + "\n" // una plantilla vacía da una nota con su título, no la nota por defecto
	}
	note, err := s.CreateNoteInDirWithBody(dir, title, body)
	if note != nil {
		note.Warnings = unknown
	}
	return note, err
}

// DailyNote devuelve la nota diaria de now (journal/AAAA-MM-DD.md) y si la acaba de crear. Si no existe la crea con la plantilla
// templates/daily.md (o, sin ella, con el título de la fecha), y crea journal/ si falta. Si ya existe, no la toca.
func (s *Storage) DailyNote(now time.Time) (*Note, bool, error) {
	name := slugDaily(config.FormatDailyName(s.dailyName(), now)) // el nombre que tendrá el archivo (CreateNoteInDirWithBody lo pasa por slug)
	path := filepath.Join(s.BaseDir, filepath.FromSlash(s.dailyRel()), name+".md")
	existing := func() (*Note, bool, error) {
		if _, err := s.ResolveNote(path); err != nil { // confinada: un enlace simbólico hacia fuera no se lee
			return nil, false, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		n := &Note{ID: filepath.Base(path), Title: name, Path: path, Content: string(data)}
		n.Tags, n.Tasks = s.extractTags(n.Content), s.extractTasks(name, path, n.Content)
		return n, false, nil
	}
	if _, err := os.Lstat(path); err == nil {
		return existing()
	}
	// la plantilla se lee y se valida antes de crear nada (ni siquiera journal/): una plantilla inválida no deja rastro
	body, unknown, err := s.renderTemplate(DailyTemplate, name, now)
	if errors.Is(err, ErrTemplateNotFound) { // sin plantilla (o quitada mientras tanto): la nota con el título de la fecha
		body, unknown, err = "# "+name+"\n\n", nil, nil
	}
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(body) == "" {
		body = "# " + name + "\n"
	}
	dir, err := s.EnsureFolder(s.dailyRel())
	if err != nil {
		return nil, false, err
	}
	note, err := s.CreateNoteInDirWithBody(dir, name, body)
	if errors.Is(err, ErrNoteExists) { // otra instancia la creó entre medio
		return existing()
	}
	if err != nil {
		return nil, false, err
	}
	note.Warnings = unknown
	return note, true, nil
}

// InTemplates dice si path está dentro de templates/ (o es esa carpeta): ahí una nota es una plantilla.
func (s *Storage) InTemplates(path string) bool { return s.inTemplates(path) }

// inTemplates dice si path está dentro de templates/ (en la raíz de la carpeta de notas).
func (s *Storage) inTemplates(path string) bool {
	rel, err := filepath.Rel(s.BaseDir, path)
	tpl := s.templatesRel()
	return err == nil && (filepath.ToSlash(rel) == tpl || strings.HasPrefix(filepath.ToSlash(rel), tpl+"/"))
}

// templateFile es el archivo de la plantilla name: el que existe con ".md" en cualquier combinación de mayúsculas (diario.MD), o name.md.
func (s *Storage) templateFile(name string) string {
	dir := filepath.Join(s.BaseDir, filepath.FromSlash(s.templatesRel()))
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(e.Name(), name+".md") {
				return filepath.Join(dir, e.Name())
			}
		}
	}
	return filepath.Join(dir, name+".md")
}

// templatesRel, dailyRel y dailyName son la carpeta de plantillas, la de las notas diarias y el formato del nombre diario con la configuración (o, sin ella o con un valor
// inválido, los de siempre). Las rutas son relativas a la carpeta de notas y van con "/".
func (s *Storage) templatesRel() string {
	if f, ok := config.CleanRelFolder(s.TemplatesFolder); ok {
		return f
	}
	return TemplatesDir
}

func (s *Storage) dailyRel() string {
	if f, ok := config.CleanRelFolder(s.DailyFolder); ok {
		return f
	}
	return JournalDir
}

func (s *Storage) dailyName() string {
	if config.ValidDailyName(s.DailyNameFormat) {
		return s.DailyNameFormat
	}
	return "YYYY-MM-DD"
}

// slugDaily es el nombre de archivo (sin .md) que sale de un nombre diario: el mismo que CreateNoteInDirWithBody le pondría.
func slugDaily(name string) string { return slug(name) }
