package links

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func targets(ls []Link) string {
	var out []string
	for _, l := range ls {
		out = append(out, fmt.Sprintf("%s|%s|%s", l.Target, l.Alias, l.Anchor))
	}
	return strings.Join(out, " ; ")
}

func TestParse(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"simple", "ver [[nota]] hoy", "nota||"},
		{"alias", "[[nota|el alias]]", "nota|el alias|"},
		{"alias con barra", "[[nota|a|b]]", "nota|a|b|"},
		{"encabezado", "[[nota#Título]]", "nota||Título"},
		{"bloque", "[[nota#^abc123]]", "nota||^abc123"},
		{"misma nota", "[[#Encabezado]]", "||Encabezado"},
		{"ruta", "[[carpeta/sub/nota]]", "carpeta/sub/nota||"},
		{"extensión", "[[nota.md]]", "nota||"},
		{"todo junto", "[[a/b#c|d]]", "a/b|d|c"},
		{"varios", "[[uno]] y [[dos|2]] y [[tres#t]]", "uno|| ; dos|2| ; tres||t"},
		{"pegados", "[[a]][[b]]", "a|| ; b||"},
		{"espacios", "[[ nota ]]", "nota||"},
		{"acentos y CJK y emoji", "[[canción]] [[日本語]] [[🙂 nota]]", "canción|| ; 日本語|| ; 🙂 nota||"},
		{"alias vacío", "[[nota|]]", "nota||"},
		{"sin destino", "[[|alias]]", ""},
		{"vacío", "[[]]", ""},
		{"solo espacios", "[[   ]]", ""},
		{"solo almohadilla", "[[#]]", ""},
		{"corchete de más a la derecha", "[[nota]]]", "nota||"},
		{"corchete de más a los lados", "[[[nota]]]", "nota||"},
		{"escapado", `\[[nota]]`, ""},
		{"embed", "![[imagen.png]]", ""},
		{"embed y enlace", "![[img.png]] y [[nota]]", "nota||"},
		{"código en línea", "`[[nota]]` y [[otra]]", "otra||"},
		{"código en línea doble", "``a [[x]] b`` [[y]]", "y||"},
		{"enlace markdown normal", "[texto](http://x) [[nota]]", "nota||"},
		{"sin cierre", "[[nota sin cerrar", ""},
		{"tabs", "\t- [[nota]]", "nota||"},
		{"en una tarea", "- [ ] llamar a [[Ana López]] hoy", "Ana López||"},
		{"con anchor y espacios", "[[ nota # Título ]]", "nota||Título"},
	}
	for _, c := range cases {
		if got := targets(Parse(c.in)); got != c.want {
			t.Errorf("%s: %q → %q, se esperaba %q", c.name, c.in, got, c.want)
		}
	}
	// posiciones y número de línea
	ls := Parse("a\nxx [[nota|n]] yy\n")
	if len(ls) != 1 || ls[0].Line != 2 || ls[0].Start != 3 || ls[0].End != 13 || ls[0].Raw != "[[nota|n]]" {
		t.Errorf("posición: %+v", ls)
	}
	// CRLF
	if got := targets(Parse("[[a]]\r\n[[b]]\r\n")); got != "a|| ; b||" {
		t.Errorf("CRLF: %q", got)
	}
}

func TestParseSkipsFencedCode(t *testing.T) {
	doc := "[[fuera1]]\n```\n[[dentro]]\n```\n[[fuera2]]\n~~~md\n[[dentro2]]\n~~~\n    ```\n[[fuera3]]\n````\n```\n[[dentro3]]\n````\n[[fuera4]]\n"
	if got := targets(Parse(doc)); got != "fuera1|| ; fuera2|| ; fuera3|| ; fuera4||" {
		t.Errorf("bloques vallados: %q", got)
	}
	// un vallado sin cerrar tapa hasta el final
	if got := targets(Parse("[[a]]\n```\n[[b]]\n[[c]]\n")); got != "a||" {
		t.Errorf("vallado abierto: %q", got)
	}
}

func notes(base string, files map[string]string) []storage.Note {
	var out []storage.Note
	for rel, content := range files {
		title := strings.TrimSuffix(filepath.Base(rel), ".md")
		title = strings.NewReplacer("-", " ", "_", " ").Replace(title)
		out = append(out, storage.Note{Path: filepath.Join(base, filepath.FromSlash(rel)), Title: title, Content: content})
	}
	return out
}

func TestResolve(t *testing.T) {
	base := "/n"
	ix := NewIndex(base, notes(base, map[string]string{
		"mi-nota.md":         "",
		"Ana López.md":       "",
		"proyectos/plan.md":  "",
		"personal/plan.md":   "",
		"personal/diario.md": "",
		"a/b/profunda.md":    "",
		"canción.md":         "",
	}))
	from := filepath.Join(base, "personal", "diario.md")
	for _, c := range []struct{ target, want string }{
		{"mi-nota", "mi-nota.md"},
		{"MI-NOTA", "mi-nota.md"}, // sin distinguir mayúsculas
		{"Mi Nota", "mi-nota.md"}, // por el título, con espacios
		{"mi-nota.md", "mi-nota.md"},
		{"ana lópez", "Ana López.md"},
		{"canción", "canción.md"},
		{"plan", "personal/plan.md"},            // dos candidatas: gana la de la misma carpeta
		{"proyectos/plan", "proyectos/plan.md"}, // la ruta desambigua
		{"b/profunda", "a/b/profunda.md"},       // el final de la ruta
		{"a/b/profunda", "a/b/profunda.md"},
		{"profunda", "a/b/profunda.md"},
	} {
		n, ok := ix.Resolve(Link{Target: c.target}, from)
		if !ok || filepath.ToSlash(strings.TrimPrefix(n.Path, base+"/")) != c.want {
			t.Errorf("%q → %v %v, se esperaba %s", c.target, n, ok, c.want)
		}
	}
	if _, ok := ix.Resolve(Link{Target: "no-existe"}, from); ok {
		t.Error("un enlace a una nota inexistente no resuelve")
	}
	if n, ok := ix.Resolve(Link{Anchor: "Encabezado"}, from); !ok || n.Path != from {
		t.Errorf("[[#h]] es la propia nota: %v %v", n, ok)
	}
	// sin carpeta común, gana la ruta más corta y luego el orden alfabético
	ix2 := NewIndex(base, notes(base, map[string]string{"x/y/dup.md": "", "x/dup.md": "", "z/dup.md": ""}))
	if n, _ := ix2.Resolve(Link{Target: "dup"}, filepath.Join(base, "otra.md")); !strings.HasSuffix(filepath.ToSlash(n.Path), "x/dup.md") {
		t.Errorf("desempate: %s", n.Path)
	}
}

func TestBacklinks(t *testing.T) {
	base := "/n"
	ix := NewIndex(base, notes(base, map[string]string{
		"destino.md":     "# Destino\nlink propio [[destino]]\n",
		"a.md":           "uno\nver [[destino]] y [[Destino|alias]]\n```\n[[destino]]\n```\n",
		"sub/b.md":       "[[destino#sección]]\notra línea\n",
		"c.md":           "sin enlaces\n",
		"d.md":           "[[otra]]\n",
		"sub/destino.md": "homónima en otra carpeta\n",
	}))
	got := ix.Backlinks(filepath.Join(base, "destino.md"))
	var lines []string
	for _, b := range got {
		lines = append(lines, fmt.Sprintf("%s:%d", filepath.ToSlash(strings.TrimPrefix(b.Note.Path, base+"/")), b.Line))
	}
	// la propia nota no cuenta; el código tampoco; "destino" desde sub/b.md resuelve a sub/destino.md (la de su carpeta)
	if strings.Join(lines, ",") != "a.md:2,a.md:2" {
		t.Errorf("backlinks %v", lines)
	}
	got = ix.Backlinks(filepath.Join(base, "sub", "destino.md"))
	if len(got) != 1 || got[0].Note.Path != filepath.Join(base, "sub", "b.md") || got[0].Text != "[[destino#sección]]" {
		t.Errorf("backlinks de la homónima: %+v", got)
	}
}

func TestRenameEdits(t *testing.T) {
	base := "/n"
	ix := NewIndex(base, notes(base, map[string]string{
		"vieja.md":     "se menciona [[vieja]] aquí\n",
		"a.md":         "x [[vieja]] y [[Vieja|alias]] y [[vieja#Título]] y [[vieja.md]]\r\nsin\n- [ ] ver [[ vieja ]]\n",
		"sub/b.md":     "[[sub/vieja]] y [[otra]]\n",
		"sub/vieja.md": "otra homónima\n",
		"c.md":         "```\n[[vieja]]\n```\n`[[vieja]]` ![[vieja]]\n",
	}))
	edits := ix.RenameEdits(filepath.Join(base, "vieja.md"), "nueva")
	got := map[string]string{}
	for _, e := range edits {
		got[fmt.Sprintf("%s:%d", filepath.ToSlash(strings.TrimPrefix(e.Path, base+"/")), e.Line)] = e.After
	}
	want := map[string]string{
		"vieja.md:1": "se menciona [[nueva]] aquí",
		"a.md:1":     "x [[nueva]] y [[nueva|alias]] y [[nueva#Título]] y [[nueva.md]]\r",
		"a.md:3":     "- [ ] ver [[nueva]]",
	}
	if len(got) != len(want) {
		t.Errorf("ediciones: %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %q, se esperaba %q", k, got[k], v)
		}
	}
	// el enlace de ruta cambia solo el último tramo; lo de código, embeds y notas homónimas no se toca
	edits = ix.RenameEdits(filepath.Join(base, "sub", "vieja.md"), "nueva")
	if len(edits) != 1 || edits[0].After != "[[sub/nueva]] y [[otra]]" {
		t.Errorf("ruta: %+v", edits)
	}
	if e := ix.RenameEdits(filepath.Join(base, "c.md"), "z"); len(e) != 0 {
		t.Errorf("sin enlaces a c.md no hay ediciones: %+v", e)
	}
}
