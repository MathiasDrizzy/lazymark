package storage

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func tplStore(t *testing.T, files map[string]string) *Storage {
	t.Helper()
	dir := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return New(dir)
}

var tplNow = time.Date(2026, 10, 3, 9, 5, 0, 0, time.Local)

// TestTemplates (C.5): las variables {{date}}, {{time}} y {{title}} se reemplazan (todas las apariciones, sin volver a expandir el título),
// lo demás queda igual; las plantillas se listan; un nombre con ruta o que no existe no vale.
func TestTemplates(t *testing.T) {
	s := tplStore(t, map[string]string{
		"templates/reunion.md": "# {{title}}\n\nFecha: {{date}} {{time}} ({{date}})\n{{otra}} {{ title }}\n- [ ] preparar\n",
		"templates/b.md":       "x",
		"templates/sub/c.md":   "no es una plantilla (subcarpeta)",
		"templates/.oculta.md": "no",
		"templates/leeme.txt":  "no",
	})
	got, err := s.RenderTemplate("reunion", "Equipo {{date}}", tplNow)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Equipo {{date}}\n\nFecha: 2026-10-03 09:05 (2026-10-03)\n{{otra}} {{ title }}\n- [ ] preparar\n"
	if got != want {
		t.Errorf("render:\n%q\nse esperaba\n%q", got, want)
	}
	if got, _ := s.RenderTemplate("reunion.md", "T", tplNow); !strings.HasPrefix(got, "# T\n") {
		t.Errorf("con .md también: %q", got)
	}
	if names := s.Templates(); !reflect.DeepEqual(names, []string{"b", "reunion"}) {
		t.Errorf("plantillas = %v", names)
	}
	for _, bad := range []string{"", "..", "../reunion", "sub/c", "nada", `a\b`} {
		if _, err := s.RenderTemplate(bad, "T", tplNow); !errors.Is(err, ErrTemplateNotFound) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	n, err := s.CreateNoteFromTemplate(s.BaseDir, "Mi reunión", "reunion", tplNow)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(n.Path); !strings.HasPrefix(string(b), "# Mi reunión\n\nFecha: 2026-10-03 09:05") {
		t.Errorf("nota: %q", b)
	}
	if _, err := s.CreateNoteFromTemplate(s.BaseDir, "Mi reunión", "reunion", tplNow); !errors.Is(err, ErrNoteExists) {
		t.Errorf("no pisa una nota que existe: %v", err)
	}
}

// TestTemplatesAreNotNotes: las etiquetas y las casillas de una plantilla no llegan al tablero ni a las categorías; la plantilla sigue
// siendo una nota que se ve y se edita.
func TestTemplatesAreNotNotes(t *testing.T) {
	s := tplStore(t, map[string]string{
		"templates/t.md": "# {{title}}\n#molde\n- [ ] tarea del molde\n",
		"n.md":           "# N\n#real\n- [ ] tarea real\n",
	})
	notes, err := s.ListNotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("%d notas", len(notes))
	}
	for _, n := range notes {
		switch filepath.Base(n.Path) {
		case "t.md":
			if len(n.Tasks) != 0 || len(n.Tags) != 0 {
				t.Errorf("la plantilla no aporta tareas ni etiquetas: %v %v", n.Tasks, n.Tags)
			}
		case "n.md":
			if len(n.Tasks) != 1 || len(n.Tags) != 1 {
				t.Errorf("la nota normal sí: %v %v", n.Tasks, n.Tags)
			}
		}
	}
}

// TestDailyNote (C.5): crea journal/AAAA-MM-DD.md con la plantilla daily (o sin ella con el título), crea journal/ si falta, no toca una nota
// que ya existe y no escribe fuera si journal es un enlace simbólico hacia fuera.
func TestDailyNote(t *testing.T) {
	s := tplStore(t, map[string]string{"templates/daily.md": "# {{title}}\n\n## Hoy ({{time}})\n- [ ] \n"})
	n, created, err := s.DailyNote(tplNow)
	if err != nil || !created {
		t.Fatalf("%v created=%v", err, created)
	}
	if filepath.Base(n.Path) != "2026-10-03.md" || filepath.Base(filepath.Dir(n.Path)) != "journal" {
		t.Errorf("ruta %s", n.Path)
	}
	if b, _ := os.ReadFile(n.Path); string(b) != "# 2026-10-03\n\n## Hoy (09:05)\n- [ ] \n" {
		t.Errorf("contenido %q", b)
	}
	os.WriteFile(n.Path, []byte("# editada\n"), 0o644)
	again, created, err := s.DailyNote(tplNow.Add(3 * time.Hour))
	if err != nil || created || again.Path != n.Path || again.Content != "# editada\n" {
		t.Errorf("la segunda vez abre la misma sin tocarla: %v created=%v %+v", err, created, again)
	}
	// otro día, sin plantilla
	s2 := tplStore(t, nil)
	n2, created, err := s2.DailyNote(tplNow)
	if err != nil || !created {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(n2.Path); string(b) != "# 2026-10-03\n\n" {
		t.Errorf("sin plantilla: %q", b)
	}
	// journal hacia fuera
	s3 := tplStore(t, nil)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(s3.BaseDir, "journal")); err == nil {
		if _, _, err := s3.DailyNote(tplNow); err == nil {
			t.Error("journal enlazado hacia fuera debe rechazarse")
		}
		if es, _ := os.ReadDir(outside); len(es) != 0 {
			t.Errorf("se escribió fuera: %v", es)
		}
	}
}

// TestTemplateUppercaseExtension (C.5, segunda opinión): una plantilla "diario.MD" se lista y se usa igual que "diario.md".
func TestTemplateUppercaseExtension(t *testing.T) {
	s := tplStore(t, map[string]string{"templates/diario.MD": "# {{title}}\n"})
	if names := s.Templates(); !reflect.DeepEqual(names, []string{"diario"}) {
		t.Fatalf("plantillas = %v", names)
	}
	for _, name := range []string{"diario", "diario.MD", "diario.md"} {
		if got, err := s.RenderTemplate(name, "T", tplNow); err != nil || got != "# T\n" {
			t.Errorf("%q: %q %v", name, got, err)
		}
	}
}
