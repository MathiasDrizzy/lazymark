package links

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// relOf es la ruta relativa a base, con "/" (en Windows las rutas llevan "\\").
func relOf(base, p string) string {
	r, err := filepath.Rel(base, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}

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
		if !ok || relOf(base, n.Path) != c.want {
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
		lines = append(lines, fmt.Sprintf("%s:%d", relOf(base, b.Note.Path), b.Line))
	}
	// la propia nota no cuenta; el código tampoco; "destino" desde sub/b.md resuelve a sub/destino.md (la de su carpeta)
	if strings.Join(lines, ",") != "a.md:2" { // dos enlaces en la misma línea son una sola entrada
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
		got[fmt.Sprintf("%s:%d", relOf(base, e.Path), e.Line)] = e.After
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

// TestAgainstObsidianOracle: casos de borde escritos por un segundo modelo (agente de apoyo) SOLO a partir de la documentación oficial de
// Obsidian (https://obsidian.md/help/links), con los enlaces que debe reconocer cada línea; el lector de lazymark debe coincidir en cada uno.
func TestAgainstObsidianOracle(t *testing.T) {
	data, err := os.ReadFile("testdata/obsidian-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Line  string `json:"line"`
		Doc   string `json:"doc"`
		Links []struct {
			Target, Alias, Anchor string
		} `json:"links"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 30 {
		t.Fatalf("solo %d casos", len(cases))
	}
	for _, c := range cases {
		text := c.Line
		if c.Doc != "" {
			text = c.Doc
		}
		var want []string
		for _, l := range c.Links {
			// la doc dice que `[[nota]]` y `[[nota.md]]` son equivalentes (https://obsidian.md/help/links): lazymark guarda el nombre sin .md
			want = append(want, fmt.Sprintf("%s|%s|%s", strings.TrimSuffix(l.Target, ".md"), l.Alias, l.Anchor))
		}
		if got := targets(Parse(text)); got != strings.Join(want, " ; ") {
			t.Errorf("%q: lazymark lee %q, el oráculo dice %q", text, got, strings.Join(want, " ; "))
		}
	}
}

func TestPlain(t *testing.T) {
	for in, want := range map[string]string{
		"ver [[nota|esa]] y [[otra]] y [[a#b]]": "ver esa y otra y a > b",
		"sin enlaces":                           "sin enlaces",
		"`[[código]]` [[x]]":                    "`[[código]]` x",
		"primera línea\nsegunda con [[enlace|ese]] y [[otro]]\n\ntercera [[a#b]]": "primera línea\nsegunda con ese y otro\n\ntercera a > b",
		"[[uno]]\r\n[[dos|2]]\r\n": "uno\r\n2\r\n",
	} {
		if got := Plain(in); got != want {
			t.Errorf("Plain(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

// TestBacklinksScale: con miles de notas con enlaces, pedir los backlinks de muchas notas no repite el trabajo (el grafo se calcula una vez
// por índice) ni resuelve cada enlace recorriendo todas las notas.
func TestBacklinksScale(t *testing.T) {
	base := "/n"
	files := map[string]string{}
	const n = 3000
	for i := 0; i < n; i++ {
		var b strings.Builder
		b.WriteString("# nota\n")
		for k := 1; k <= 5; k++ {
			fmt.Fprintf(&b, "enlace a [[nota-%d]] y [[carpeta-%d/nota-%d|alias]]\n", (i*7+k*13)%n, (i+k)%30, (i*7+k*13)%n)
		}
		files[fmt.Sprintf("carpeta-%d/nota-%d.md", i%30, i)] = b.String()
	}
	start := time.Now()
	ix := NewIndex(base, notes(base, files))
	total := 0
	for i := 0; i < 300; i++ {
		total += len(ix.Backlinks(filepath.Join(base, fmt.Sprintf("carpeta-%d", i%30), fmt.Sprintf("nota-%d.md", i))))
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("3000 notas con 30000 enlaces y 300 consultas de backlinks tardaron %v: algo es cuadrático", d)
	}
	if total == 0 {
		t.Error("el grafo no halló ningún backlink")
	}
}

// TestRenameEditsOnCRLFNote: las ediciones de una nota con CRLF conservan el \r (y ReplaceLineIf las aplica contra la misma línea).
func TestRenameEditsOnCRLFNote(t *testing.T) {
	base := "/n"
	ix := NewIndex(base, notes(base, map[string]string{"vieja.md": "x\n", "a.md": "uno\r\nver [[vieja|v]] fin\r\n"}))
	e := ix.RenameEdits(filepath.Join(base, "vieja.md"), "nueva")
	if len(e) != 1 || e[0].Before != "ver [[vieja|v]] fin\r" || e[0].After != "ver [[nueva|v]] fin\r" || e[0].Line != 2 {
		t.Errorf("%+v", e)
	}
}

// TestFolderRenameEdits (ORD-017 F6 / L5): al renombrar una carpeta se actualizan los wikilinks de ruta que pasan por ella (con su alias, anchor y .md);
// los de solo nombre no cambian porque siguen resolviendo; los que apuntan a notas de fuera tampoco.
func TestFolderRenameEdits(t *testing.T) {
	base := filepath.FromSlash("/n")
	files := map[string]string{
		"proyectos/plan.md":      "# plan\n",
		"proyectos/sub/deep.md":  "# deep\n",
		"fuera/plan.md":          "otra con el mismo nombre\n",
		"otra.md":                "a [[proyectos/plan]] b [[proyectos/plan|el alias]] c [[proyectos/plan.md#Meta]]\nname [[plan]]  [[sub/deep]]  [[proyectos/sub/deep#h|x]]\nfuera [[fuera/plan]]\n",
		"proyectos/interno.md":   "dentro [[proyectos/plan]] y [[sub/deep]] y [[../fuera/plan]]\r\n",
		"dos/proyectos/otra2.md": "[[proyectos/otra2]] es otra carpeta\n",
	}
	ix := NewIndex(base, notes(base, files))
	apply := func(edits []Edit) map[string]string {
		out := map[string]string{}
		for _, e := range edits {
			out[relOf(base, e.Path)+":"+itoa(e.Line)] = e.After
		}
		return out
	}
	got := apply(ix.FolderRenameEdits(filepath.Join(base, "proyectos"), "trabajos"))
	want := map[string]string{
		"otra.md:1":              "a [[trabajos/plan]] b [[trabajos/plan|el alias]] c [[trabajos/plan.md#Meta]]",
		"otra.md:2":              "name [[plan]]  [[sub/deep]]  [[trabajos/sub/deep#h|x]]",
		"proyectos/interno.md:1": "dentro [[trabajos/plan]] y [[sub/deep]] y [[../fuera/plan]]\r",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s:\n got %q\nwant %q", k, got[k], w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("solo deben cambiar %d líneas, cambian %d: %v", len(want), len(got), got)
	}
	// una subcarpeta: el tramo del medio
	got = apply(ix.FolderRenameEdits(filepath.Join(base, "proyectos", "sub"), "hijo"))
	if g := got["otra.md:2"]; g != "name [[plan]]  [[hijo/deep]]  [[proyectos/hijo/deep#h|x]]" || got["proyectos/interno.md:1"] != "dentro [[proyectos/plan]] y [[hijo/deep]] y [[../fuera/plan]]\r" || len(got) != 2 {
		t.Errorf("subcarpeta: %v", got)
	}
	// una carpeta sin enlaces de ruta: nada que actualizar
	if e := ix.FolderRenameEdits(filepath.Join(base, "fuera"), "otro"); len(e) != 1 || !strings.Contains(e[0].After, "[[otro/plan]]") {
		t.Errorf("fuera/: %v", e)
	}
}

// TestFolderRenameEditsWindowsSeparators (ORD-017, segunda opinión): un enlace escrito con barra invertida ([[carpeta\nota]]) resuelve (key lo normaliza) y también se
// actualiza al renombrar la carpeta; renombrar una nota con ese enlace conserva su carpeta.
func TestFolderRenameEditsWindowsSeparators(t *testing.T) {
	base := filepath.FromSlash("/n")
	ix := NewIndex(base, notes(base, map[string]string{"carpeta/nota.md": "# n\n", "hub.md": "ver [[carpeta\\nota]]\n"}))
	e := ix.FolderRenameEdits(filepath.Join(base, "carpeta"), "trabajos")
	if len(e) != 1 || e[0].After != "ver [[trabajos/nota]]" {
		t.Errorf("carpeta: %+v", e)
	}
	e = ix.RenameEdits(filepath.Join(base, "carpeta", "nota.md"), "nueva")
	if len(e) != 1 || e[0].After != "ver [[carpeta\\nueva]]" {
		t.Errorf("nota: %+v", e)
	}
}

// TestFolderRenameHomonymsAndCode (ORD-017 rev 2, R2-4): con dos carpetas homónimas (a/docs y b/docs), renombrar a/docs no toca los links a b/docs; y los wikilinks
// dentro de bloques de código (o código en línea) no se reescriben.
func TestFolderRenameHomonymsAndCode(t *testing.T) {
	base := filepath.FromSlash("/n")
	ix := NewIndex(base, notes(base, map[string]string{
		"a/docs/x.md": "# x\n",
		"b/docs/y.md": "# y\n",
		"hub.md":      "[[a/docs/x]] y [[b/docs/y]]\n```\n[[a/docs/x]]\n```\ncódigo `[[a/docs/x]]` aquí\n",
	}))
	e := ix.FolderRenameEdits(filepath.Join(base, "a", "docs"), "guias")
	if len(e) != 1 || e[0].After != "[[a/guias/x]] y [[b/docs/y]]" {
		t.Errorf("solo el link a a/docs cambia: %+v", e)
	}
}
