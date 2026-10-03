package storage

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Columns son los ids de las columnas del tablero Kanban, en orden. La primera es donde cae una tarea sin tag;
// la de hecho es la que se llama "done" o, si no hay, la última. La columna de una tarea se guarda como un tag
// al final de su línea (`- [ ] tarea #kb/doing`); `[x]` siempre cuenta como hecha. Solo se escriben `[ ]` y `[x]`
// (`[/]` y `[-]` no son GFM: GitHub no los muestra como casilla, docs/research/kanban-markdown.md).
type Columns []string

// DefaultColumns son las columnas por defecto.
var DefaultColumns = Columns{"todo", "doing", "done"}

// legacyDoing son las etiquetas con las que las versiones anteriores marcaban "en progreso". Se leen como
// #kb/doing y se migran a #kb/doing solo al mover la tarjeta, nunca al leer.
var legacyDoing = map[string]bool{"#doing": true, "#wip": true, "#progreso": true, "#in-progress": true}

// DoneIndex devuelve la columna de las tareas hechas.
func (c Columns) DoneIndex() int {
	if i := c.Index("done"); i >= 0 {
		return i
	}
	return len(c) - 1
}

// Index devuelve la posición del id (sin distinguir mayúsculas), o -1.
func (c Columns) Index(id string) int {
	for i, x := range c {
		if strings.EqualFold(x, id) {
			return i
		}
	}
	return -1
}

// tagHit es una etiqueta de control del tablero dentro de un texto: #kb/<id> o una del formato anterior.
type tagHit struct {
	start, end int // el span de la etiqueta, sin el espacio que la precede
	id         string
	legacy     bool
}

var tokenRe = regexp.MustCompile(`#[A-Za-z0-9_/-]+`)

// scanTags devuelve las etiquetas de control de text: las que empiezan el texto o van tras un espacio.
func scanTags(text string) []tagHit {
	var hits []tagHit
	for _, loc := range tokenRe.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && text[loc[0]-1] != ' ' && text[loc[0]-1] != '\t' {
			continue
		}
		tok := strings.ToLower(text[loc[0]:loc[1]])
		switch {
		case strings.HasPrefix(tok, "#kb/") && len(tok) > 4 && !strings.ContainsRune(tok[4:], '/'):
			hits = append(hits, tagHit{start: loc[0], end: loc[1], id: tok[4:]})
		case legacyDoing[tok]:
			hits = append(hits, tagHit{start: loc[0], end: loc[1], id: "doing", legacy: true})
		}
	}
	return hits
}

// Of devuelve la columna de la tarea: la de hecho si está marcada; si no, la del primer #kb/<id> (una que no
// existe cae en la primera); si no, la de una etiqueta del formato anterior (#doing…) como doing; si no, la primera.
func (c Columns) Of(t Task) int {
	if t.Done {
		return c.DoneIndex()
	}
	hits := scanTags(t.Text)
	for _, h := range hits {
		if !h.legacy {
			if i := c.Index(h.id); i >= 0 && i != c.DoneIndex() {
				return i
			}
			return 0
		}
	}
	for _, h := range hits {
		if i := c.Index(h.id); i >= 0 {
			return i
		}
		return 0
	}
	return 0
}

// CleanTaskText devuelve el texto de la tarea sin lo que es del tablero y de las fechas: las etiquetas #kb/<col> (y las
// #doing… heredadas) y la primera fecha válida de cada campo (🛫 📅 ✅). Las demás etiquetas y las fechas inválidas o
// repetidas se quedan.
func CleanTaskText(text string) string {
	type span struct{ start, end int }
	var spans []span
	for _, h := range scanTags(text) {
		spans = append(spans, span{h.start, h.end})
	}
	seen := map[DateField]bool{}
	for _, h := range scanDates(text) {
		if !seen[h.field] {
			seen[h.field] = true
			spans = append(spans, span{h.start, h.end})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start > spans[j].start })
	for _, sp := range spans {
		text = removeSpan(text, sp.start, sp.end)
	}
	return strings.Join(strings.Fields(text), " ")
}

// removeSpan quita text[start:end] con los espacios que lo preceden.
func removeSpan(text string, start, end int) string {
	for start > 0 && (text[start-1] == ' ' || text[start-1] == '\t') {
		start--
	}
	return text[:start] + text[end:]
}

// removeHit quita una etiqueta con los espacios que la preceden.
func removeHit(text string, h tagHit) string {
	start := h.start
	for start > 0 && (text[start-1] == ' ' || text[start-1] == '\t') {
		start--
	}
	return text[:start] + text[h.end:]
}

// RewriteForColumn devuelve la línea de tarea con la casilla y el tag de la columna target: a la de hecho, `[x]` y sin
// tag (y con `✅ hoy` si no tenía fecha de completada); a la primera, `[ ]` y sin tag; a las demás, `[ ]` y `#kb/<id>`
// (en el lugar del tag que ya había, o al final). Al salir de la columna de hecho se quita el `✅`. Las etiquetas del
// formato anterior se reemplazan por la nueva. No toca nada más de la línea.
func RewriteForColumn(line string, cols Columns, target int) (string, error) {
	if target < 0 || target >= len(cols) {
		return "", fmt.Errorf("la columna %d no existe (hay %d)", target, len(cols))
	}
	m := toggleTaskRegex.FindStringSubmatch(line)
	if len(m) != 4 {
		return "", fmt.Errorf("la línea no es una tarea válida de markdown")
	}
	cr := ""
	if strings.HasSuffix(m[3], "\r") {
		cr, m[3] = "\r", strings.TrimSuffix(m[3], "\r")
	}
	mark := m[2]
	wasDone := mark == "x" || mark == "X"
	switch {
	case target == cols.DoneIndex():
		if mark != "x" && mark != "X" {
			mark = "x"
		}
	default:
		mark = " "
	}
	rest := m[3][1:] // lo que sigue al "]"
	hits := scanTags(rest)
	tag := ""
	if target != 0 && target != cols.DoneIndex() {
		tag = "#kb/" + cols[target]
	}
	// la que se reemplaza en su sitio: el primer #kb/, o si no el primero del formato anterior
	keep := -1
	for i, h := range hits {
		if !h.legacy {
			keep = i
			break
		}
	}
	if keep < 0 && len(hits) > 0 {
		keep = 0
	}
	for i := len(hits) - 1; i >= 0; i-- {
		switch {
		case i == keep && tag != "":
			rest = rest[:hits[i].start] + tag + rest[hits[i].end:]
		default:
			rest = removeHit(rest, hits[i])
		}
	}
	if tag != "" && keep < 0 {
		trimmed := strings.TrimRight(rest, " \t")
		rest = trimmed + " " + tag + rest[len(trimmed):]
	}
	out := m[1] + mark + "]" + rest + cr
	if wasDone && target == cols.DoneIndex() {
		return out, nil // ya estaba hecha: no se cambia su fecha de completada
	}
	return withCompletion(out, target == cols.DoneIndex()), nil
}

// MoveTask mueve la tarea de la línea line de la nota a la columna target, reescribiendo solo esa línea. Si
// expected no es cero y la nota cambió en disco desde que se leyó, no escribe y devuelve ErrNoteChanged (X10).
func (s *Storage) MoveTask(notePath string, line int, cols Columns, target int, expected time.Time) error {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	return rewriteLine(notePath, line, expected, func(l string) (string, error) {
		out, err := RewriteForColumn(l, cols, target)
		if err != nil {
			return "", fmt.Errorf("la línea %d: %w", line, err)
		}
		return out, nil
	})
}

// withCompletion mantiene la fecha de completada de una tarea según su casilla: al quedar hecha se agrega `✅ hoy` si no
// tenía una válida (si ya la tenía, por ejemplo de Obsidian, se respeta), y al dejar de estarlo se quita.
func withCompletion(line string, done bool) string {
	has := false
	for _, h := range scanDates(line) {
		if h.field == DateDone {
			has = true
		}
	}
	switch {
	case done && !has:
		return setDate(line, DateDone, Today())
	case !done && has:
		for { // también las repetidas: si no, quedaría una fecha de completada en una tarea abierta
			hasMore := false
			for _, h := range scanDates(line) {
				if h.field == DateDone {
					hasMore = true
				}
			}
			if !hasMore {
				return line
			}
			line = setDate(line, DateDone, "")
		}
	}
	return line
}
