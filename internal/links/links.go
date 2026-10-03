// Package links entiende los wikilinks de las notas: `[[nota]]`, `[[nota|alias]]`, `[[nota#Encabezado]]`, `[[nota#^bloque]]`, `[[#Encabezado]]` y
// `[[carpeta/nota]]`, como en Obsidian (https://help.obsidian.md/links). Lee los enlaces de un texto, los resuelve a una nota, calcula los
// backlinks (qué notas enlazan a una) y las ediciones que hacen falta para que los enlaces sigan valiendo cuando se renombra una nota.
package links

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// Link es un wikilink dentro de un texto.
type Link struct {
	Line   int    // desde 1
	Start  int    // byte de la línea donde empieza el `[[`
	End    int    // byte donde termina (después del `]]`)
	Target string // el nombre de la nota, sin alias, sin # y sin .md ("" si es un enlace a un encabezado de la misma nota)
	Alias  string // el texto a mostrar ("" si no hay)
	Anchor string // lo que va tras # ("Encabezado" o "^bloque"), o ""
	Raw    string // el enlace tal como está escrito, con los corchetes
}

// Display es el texto con el que se muestra el enlace: el alias si lo hay, si no el nombre (con su anchor).
func (l Link) Display() string {
	switch {
	case l.Alias != "":
		return l.Alias
	case l.Target != "" && l.Anchor != "":
		return l.Target + " > " + l.Anchor
	case l.Target != "":
		return l.Target
	}
	return l.Anchor
}

// Parse devuelve los wikilinks de content, en orden. No cuentan: los embeds (`![[…]]`), los escapados (`\[[…]]`), los vacíos, los que
// están dentro de código en línea o de bloques de código vallados.
func Parse(content string) []Link {
	var out []Link
	fence := "" // el vallado abierto (``` o ~~~, con su largo), o ""
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if f := fenceOf(line); f != "" {
			switch {
			case fence == "":
				fence = f
			case f[0] == fence[0] && len(f) >= len(fence) && strings.TrimSpace(line)[len(f):] == "":
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		out = append(out, parseLine(line, n+1)...)
	}
	return out
}

// fenceOf devuelve el vallado (3 o más ` o ~) con el que empieza la línea (hasta 3 espacios de sangría), o "".
func fenceOf(line string) string {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 || len(t) < 3 || (t[0] != '`' && t[0] != '~') {
		return ""
	}
	i := 0
	for i < len(t) && t[i] == t[0] {
		i++
	}
	if i < 3 || (t[0] == '`' && strings.Contains(t[i:], "`")) {
		return ""
	}
	return t[:i]
}

// codeMask marca los bytes de la línea que están dentro de código en línea (entre rachas de comillas inversas del mismo largo).
func codeMask(line string) []bool {
	mask := make([]bool, len(line))
	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		j := i
		for j < len(line) && line[j] == '`' {
			j++
		}
		run := line[i:j]
		if k := strings.Index(line[j:], run); k >= 0 { // el cierre: una racha igual (si es más larga no cierra, pero es raro; se acepta)
			for x := i; x < j+k+len(run); x++ {
				mask[x] = true
			}
			i = j + k + len(run)
			continue
		}
		i = j
	}
	return mask
}

func parseLine(line string, n int) []Link {
	if !strings.Contains(line, "[[") {
		return nil
	}
	mask := codeMask(line)
	var out []Link
	for i := 0; i+1 < len(line); {
		if line[i] != '[' || line[i+1] != '[' || mask[i] {
			i++
			continue
		}
		if i > 0 && line[i-1] == '\\' { // `\[[nota]]` está escapado
			i += 2
			continue
		}
		end := strings.Index(line[i+2:], "]]")
		if end < 0 {
			break
		}
		inner := line[i+2 : i+2+end]
		after := i + 2 + end + 2
		if i > 0 && line[i-1] == '!' { // un embed (`![[archivo]]`) no es un enlace normal
			i = after
			continue
		}
		if strings.ContainsAny(inner, "[]") || containsMasked(mask, i, after) {
			i++ // p. ej. `[[[nota]]]`: el enlace empieza un byte más adelante
			continue
		}
		if l, ok := build(inner, n, i, after, line[i:after]); ok {
			out = append(out, l)
		}
		i = after
	}
	return out
}

func containsMasked(mask []bool, from, to int) bool {
	for x := from; x < to && x < len(mask); x++ {
		if mask[x] {
			return true
		}
	}
	return false
}

// build arma un enlace a partir de lo que hay entre los corchetes; no es un enlace si queda vacío o sin destino ni encabezado.
func build(inner string, line, start, end int, raw string) (Link, bool) {
	target, alias, hasAlias := strings.Cut(inner, "|")
	target = strings.TrimSpace(target)
	if target == "" && !hasAlias && strings.TrimSpace(inner) == "" {
		return Link{}, false
	}
	name, anchor, _ := strings.Cut(target, "#")
	name = strings.TrimSuffix(strings.TrimSpace(name), ".md")
	anchor = strings.TrimSpace(anchor)
	if name == "" && anchor == "" { // `[[|alias]]`, `[[#]]`
		return Link{}, false
	}
	return Link{Line: line, Start: start, End: end, Target: name, Alias: strings.TrimSpace(alias), Anchor: anchor, Raw: raw}, true
}

// ─── resolución ────────────────────────────────────────────────────

// key normaliza un nombre para compararlo: minúsculas, "/" como separador y sin .md.
func key(s string) string {
	return strings.ToLower(strings.TrimSuffix(strings.ReplaceAll(strings.TrimSpace(s), "\\", "/"), ".md"))
}

// Index permite resolver enlaces contra las notas de una carpeta.
type Index struct {
	base  string
	notes []storage.Note
	rel   []string // la ruta relativa de cada nota, normalizada con key
}

// NewIndex arma el índice de resolución de las notas (con su carpeta base).
func NewIndex(base string, notes []storage.Note) *Index {
	ix := &Index{base: base, notes: notes, rel: make([]string, len(notes))}
	for i, n := range notes {
		r, err := filepath.Rel(base, n.Path)
		if err != nil {
			r = n.Path
		}
		ix.rel[i] = key(filepath.ToSlash(r))
	}
	return ix
}

// Resolve devuelve la nota a la que apunta el enlace, escrito en la nota from. Sin sin nombre (`[[#h]]`) es la propia nota. Con "/" se
// compara contra la ruta relativa (o su final); sin ella, contra el nombre del archivo y, si no, contra el título (con espacios).
// Con varias candidatas gana la de la misma carpeta que from, luego la de ruta más corta y luego la primera por orden alfabético.
func (ix *Index) Resolve(l Link, from string) (*storage.Note, bool) {
	if l.Target == "" {
		for i := range ix.notes {
			if ix.notes[i].Path == from {
				return &ix.notes[i], true
			}
		}
		return nil, false
	}
	want := key(l.Target)
	var cands []int
	for i := range ix.notes {
		r := ix.rel[i]
		switch {
		case strings.Contains(want, "/"):
			if r == want || strings.HasSuffix(r, "/"+want) {
				cands = append(cands, i)
			}
		default:
			base := r[strings.LastIndex(r, "/")+1:]
			title := strings.ToLower(ix.notes[i].Title)
			if base == want || title == want {
				cands = append(cands, i)
			}
		}
	}
	if len(cands) == 0 {
		return nil, false
	}
	fromDir := filepath.Dir(from)
	sort.SliceStable(cands, func(a, b int) bool {
		na, nb := ix.notes[cands[a]], ix.notes[cands[b]]
		sa, sb := filepath.Dir(na.Path) == fromDir, filepath.Dir(nb.Path) == fromDir
		if sa != sb {
			return sa
		}
		if la, lb := len(ix.rel[cands[a]]), len(ix.rel[cands[b]]); la != lb {
			return la < lb
		}
		return ix.rel[cands[a]] < ix.rel[cands[b]]
	})
	return &ix.notes[cands[0]], true
}

// Backlink es una línea de otra nota que enlaza a la nota pedida.
type Backlink struct {
	Note *storage.Note
	Line int
	Text string // la línea, sin sangría
}

// Backlinks devuelve las líneas de las demás notas que enlazan a la nota to, ordenadas por nota y línea.
func (ix *Index) Backlinks(to string) []Backlink {
	var out []Backlink
	for i := range ix.notes {
		n := &ix.notes[i]
		if n.Path == to || !strings.Contains(n.Content, "[[") {
			continue
		}
		lines := strings.Split(n.Content, "\n")
		for _, l := range Parse(n.Content) {
			if dst, ok := ix.Resolve(l, n.Path); ok && dst.Path == to {
				out = append(out, Backlink{Note: n, Line: l.Line, Text: strings.TrimSpace(strings.TrimSuffix(lines[l.Line-1], "\r"))})
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Note.Path != out[b].Note.Path {
			return out[a].Note.Path < out[b].Note.Path
		}
		return out[a].Line < out[b].Line
	})
	return out
}

// ─── renombrar ─────────────────────────────────────────────────────

// Edit es un cambio de una línea: lo que hay y lo que debe quedar.
type Edit struct {
	Path          string
	Line          int
	Before, After string
}

// RenameEdits calcula las líneas que hay que cambiar para que los enlaces a la nota old sigan valiendo si pasa a llamarse newBase (el
// nombre del archivo sin .md). Los enlaces de ruta (`[[carpeta/nota]]`) cambian solo el último tramo; el alias, el # y el estilo de
// mayúsculas del resto del enlace se conservan. Hay que llamarla ANTES de renombrar (resuelve con los nombres de ahora).
func (ix *Index) RenameEdits(old, newBase string) []Edit {
	byLine := map[string][]Link{} // "ruta\x00línea" → enlaces de esa línea que apuntan a old
	var order []string
	for i := range ix.notes {
		n := &ix.notes[i]
		if !strings.Contains(n.Content, "[[") {
			continue
		}
		for _, l := range Parse(n.Content) {
			if dst, ok := ix.Resolve(l, n.Path); ok && dst.Path == old && l.Target != "" {
				k := n.Path + "\x00" + itoa(l.Line)
				if _, seen := byLine[k]; !seen {
					order = append(order, k)
				}
				byLine[k] = append(byLine[k], l)
			}
		}
	}
	var edits []Edit
	for _, k := range order {
		path, lineNo, _ := strings.Cut(k, "\x00")
		var content string
		for i := range ix.notes {
			if ix.notes[i].Path == path {
				content = ix.notes[i].Content
			}
		}
		lines := strings.Split(content, "\n")
		n := atoi(lineNo)
		before := lines[n-1]
		after := before
		ls := byLine[k]
		sort.Slice(ls, func(a, b int) bool { return ls[a].Start > ls[b].Start }) // de atrás hacia adelante: los índices anteriores siguen valiendo
		cr := strings.HasSuffix(after, "\r")
		after = strings.TrimSuffix(after, "\r")
		for _, l := range ls {
			after = after[:l.Start] + rewrite(l, newBase) + after[l.End:]
		}
		if cr {
			after += "\r"
		}
		if after != before {
			edits = append(edits, Edit{Path: path, Line: n, Before: before, After: after})
		}
	}
	sort.SliceStable(edits, func(a, b int) bool {
		if edits[a].Path != edits[b].Path {
			return edits[a].Path < edits[b].Path
		}
		return edits[a].Line < edits[b].Line
	})
	return edits
}

// rewrite devuelve el enlace l apuntando a newBase, con su alias y su anchor.
func rewrite(l Link, newBase string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(l.Raw, "[["), "]]")
	targetPart, alias, hasAlias := strings.Cut(inner, "|")
	nameAnchor := strings.TrimSpace(targetPart)
	_, anchor, hasAnchor := strings.Cut(nameAnchor, "#")
	name := nameAnchor
	if i := strings.Index(nameAnchor, "#"); i >= 0 {
		name = nameAnchor[:i]
	}
	name = strings.TrimSpace(name)
	hadMD := strings.HasSuffix(strings.ToLower(name), ".md")
	dir := ""
	if i := strings.LastIndex(name, "/"); i >= 0 {
		dir = name[:i+1]
	}
	out := dir + newBase
	if hadMD {
		out += ".md"
	}
	if hasAnchor {
		out += "#" + anchor
	}
	if hasAlias {
		out += "|" + alias
	}
	return "[[" + out + "]]"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
