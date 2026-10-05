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

// TestInvalidTemplatesAreRejected (ORD-014 M1): una plantilla que no es texto (bytes nulos, UTF-16, UTF-8 inválido) o pesa más de 1 MB se
// rechaza con ErrTemplateInvalid y no se crea nada; una con BOM UTF-8 vale y la nota no lleva el BOM.
func TestInvalidTemplatesAreRejected(t *testing.T) {
	utf16 := []byte{0xff, 0xfe, '#', 0, ' ', 0, 'h', 0, 'i', 0, '\n', 0}
	s := tplStore(t, map[string]string{
		"templates/nulos.md":  "# titulo\x00con nulo\n",
		"templates/latin1.md": "caf\xe9 \xff\xfe\n",
		"templates/ok.md":     "\xef\xbb\xbf# {{title}}\n",
		"templates/enorme.md": "# x\n" + strings.Repeat("0123456789abcdef", 70<<10), // ~1,1 MB, bastante más que el tope
	})
	os.WriteFile(filepath.Join(s.BaseDir, "templates", "utf16.md"), utf16, 0o644)
	for _, name := range []string{"nulos", "latin1", "utf16", "enorme"} {
		if _, err := s.RenderTemplate(name, "T", tplNow); !(err != nil && strings.Contains(err.Error(), "plantilla") && !errors.Is(err, ErrTemplateNotFound)) {
			t.Errorf("%s: se esperaba ErrTemplateInvalid, dio %v", name, err)
		}
		if _, err := s.CreateNoteFromTemplate(s.BaseDir, "Nota "+name, name, tplNow); !(err != nil && strings.Contains(err.Error(), "plantilla") && !errors.Is(err, ErrTemplateNotFound)) {
			t.Errorf("%s: crear debe rechazarse: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(s.BaseDir, "nota-"+name+".md")); err == nil {
			t.Errorf("%s: no debía crearse la nota", name)
		}
	}
	n, err := s.CreateNoteFromTemplate(s.BaseDir, "Con bom", "ok", tplNow)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(n.Path); string(b) != "# Con bom\n" {
		t.Errorf("el BOM no debe pasar a la nota: %q", b)
	}
}

// TestSwapNumberedWithTabs (ORD-014 M3): al reordenar ítems numerados (9. ↔ 10.) las líneas sangradas con tab no se tocan (un espacio antes
// de un tab no cambia nada y ensucia el archivo), las de espacios se corren con el número y las mezcladas siguen dentro del ítem.
func TestSwapNumberedWithTabs(t *testing.T) {
	cases := map[string]struct{ doc, want string }{
		"tabs": {
			"9. [ ] a\n\t- [ ] sa\n10. [ ] b\n\t- [ ] sb\n",
			"9. [ ] b\n\t- [ ] sb\n10. [ ] a\n\t- [ ] sa\n",
		},
		"espacios": {
			"9. [ ] a\n   - [ ] sa\n10. [ ] b\n    - [ ] sb\n",
			"9. [ ] b\n   - [ ] sb\n10. [ ] a\n    - [ ] sa\n",
		},
		"mezcla": {
			"9. [ ] a\n   - [ ] s1\n\t- [ ] s2\n  \t- [ ] s3\n10. [ ] b\n    texto\n\tmás\n",
			"9. [ ] b\n   texto\n\tmás\n10. [ ] a\n    - [ ] s1\n\t - [ ] s2\n  \t - [ ] s3\n",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s, p := kanbanNote(t, c.doc)
			n := strings.Count(strings.SplitAfter(c.doc, "10.")[0], "\n") + 1 // la línea del ítem 10
			if _, _, err := s.SwapTasks(p, 1, n, time.Time{}); err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(p)
			if string(b) != c.want {
				t.Errorf("\n got %q\nwant %q", b, c.want)
			}
			if got, want := itemsHTML(t, string(b)), itemsHTML(t, c.doc); len(got) != len(want) {
				t.Errorf("cambió la estructura de ítems: %d → %d", len(want), len(got))
			}
		})
	}
}

// TestTemplateVariables (ORD-014 M2): los nombres conocidos no distinguen mayúsculas ({{Date}}, {{TITLE}}); las desconocidas ({{fecha}}, con
// espacios dentro, anidadas) quedan tal cual y se devuelven como aviso, sin repetir y en orden; la nota se crea igual con ellas.
func TestTemplateVariables(t *testing.T) {
	s := tplStore(t, map[string]string{
		"templates/v.md": "{{Date}} {{TIME}} {{Title}}\n{{fecha}} {{ title }} {{fecha}} {{{date}}} {{{{title}}}}\n",
	})
	out, unknown, err := s.renderTemplate("v", "T", tplNow)
	if err != nil {
		t.Fatal(err)
	}
	want := "2026-10-03 09:05 T\n{{fecha}} {{ title }} {{fecha}} {2026-10-03} {{T}}\n"
	if out != want {
		t.Errorf("render:\n%q\nse esperaba\n%q", out, want)
	}
	if !reflect.DeepEqual(unknown, []string{"{{fecha}}", "{{ title }}"}) {
		t.Errorf("desconocidas = %q", unknown)
	}
	n, err := s.CreateNoteFromTemplate(s.BaseDir, "Con vars", "v", tplNow)
	if err != nil || !reflect.DeepEqual(n.Warnings, []string{"{{fecha}}", "{{ title }}"}) {
		t.Fatalf("la nota se crea con aviso: %v %v", err, n)
	}
	if b, _ := os.ReadFile(n.Path); !strings.Contains(string(b), "{{fecha}}") {
		t.Errorf("la variable desconocida queda literal: %q", b)
	}
	// sin desconocidas, sin avisos
	s2 := tplStore(t, map[string]string{"templates/ok.md": "{{date}} {{title}}\n"})
	if n, err := s2.CreateNoteFromTemplate(s2.BaseDir, "A", "ok", tplNow); err != nil || len(n.Warnings) != 0 {
		t.Errorf("sin avisos: %v %v", err, n)
	}
}

// TestTemplateSizeLimit (ORD-014): el tope es 256 KB exactos: una plantilla de justo 256 KB vale y una de 256 KB + 1 byte se rechaza.
func TestTemplateSizeLimit(t *testing.T) {
	s := tplStore(t, map[string]string{
		"templates/justo.md":  strings.Repeat("a", maxTemplateBytes),
		"templates/pasada.md": strings.Repeat("a", maxTemplateBytes+1),
	})
	if got, err := s.RenderTemplate("justo", "T", tplNow); err != nil || len(got) != maxTemplateBytes {
		t.Errorf("256 KB exactos debe valer: %d %v", len(got), err)
	}
	if _, err := s.RenderTemplate("pasada", "T", tplNow); !errors.Is(err, ErrTemplateInvalid) {
		t.Errorf("256 KB + 1 debe rechazarse: %v", err)
	}
}
