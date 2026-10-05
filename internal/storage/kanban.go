package storage

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
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
	if len(hits) > 0 { // solo quedan las del formato anterior, y todas son "doing": basta mirar la primera
		if i := c.Index(hits[0].id); i >= 0 {
			return i
		}
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

// removeSpan quita text[start:end] y normaliza los espacios de alrededor: lo que queda a los lados se une con un solo espacio
// (nunca pegado), y si lo de la derecha es solo espacio final (o el \r de una línea CRLF) se conserva tal cual.
func removeSpan(text string, start, end int) string {
	left := strings.TrimRightFunc(text[:start], unicode.IsSpace)
	right := text[end:]
	if strings.TrimSpace(right) == "" {
		return left + right
	}
	right = strings.TrimLeftFunc(right, unicode.IsSpace) // también el espacio ideográfico (U+3000) del japonés y el chino
	if left == "" {
		return right
	}
	return left + " " + right
}

// removeHit quita una etiqueta del tablero y normaliza los espacios (ver removeSpan).
func removeHit(text string, h tagHit) string { return removeSpan(text, h.start, h.end) }

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
		return setDate(line, DateDone, "") // quita todos los ✅
	}
	return line
}

// ErrNotSiblings es el error de SwapTasks cuando las dos tareas no se pueden intercambiar: no son hermanas (otra sangría), una cuelga
// de la otra o alguna de las líneas ya no es una tarea.
var ErrNotSiblings = errors.New("las tareas no son hermanas")

// itemMarker reconoce el comienzo de un ítem de lista (viñeta o número) y devuelve el ancho donde empieza su contenido, que es la sangría
// que deben tener las líneas siguientes para pertenecerle (CommonMark 5.2).
var itemMarker = regexp.MustCompile(`^(\s*)((?:[-*+])|(?:\d{1,9}[.)]))( +|\t|$)`)

// contentIndent es la sangría del contenido del ítem de la línea l (la de su viñeta o número más el espacio que le sigue; con 5 o más
// espacios, solo uno cuenta). 0 si l no es un ítem.
func contentIndent(l string) int {
	m := itemMarker.FindStringSubmatch(strings.TrimSuffix(l, "\r"))
	if m == nil {
		return 0
	}
	col := indentWidth(m[1]) + len(m[2]) // columna donde empieza el espacio que sigue al marcador (un tab llega a la siguiente parada de 4)
	sp := 0
	for _, r := range m[3] {
		if r == '\t' {
			sp += 4 - (col+sp)%4
		} else {
			sp++
		}
	}
	if sp == 0 || sp > 4 {
		sp = 1
	}
	return col + sp
}

// taskBlock devuelve el rango [start, end] (índices de lines) de la tarea de la línea idx: lo que le pertenece según CommonMark, o sea su
// línea y las siguientes con la sangría de su contenido o más (continuaciones, sublistas, bloques de código), también si hay líneas en
// blanco de por medio. Las líneas en blanco del final no son del bloque. Un vallado de código dentro del ítem termina con él.
func taskBlock(lines []string, idx int) (int, int) {
	ci := contentIndent(lines[idx])
	if ci == 0 {
		return idx, idx
	}
	end := idx
	for i := idx + 1; i < len(lines); i++ {
		l := strings.TrimSuffix(lines[i], "\r")
		if strings.TrimSpace(l) == "" {
			continue // un blanco pertenece al ítem solo si sigue algo que también le pertenece
		}
		if indentWidth(leadingSpace(l)) < ci {
			break
		}
		end = i
	}
	return idx, end
}

func leadingSpace(l string) string { return l[:len(l)-len(strings.TrimLeft(l, " \t"))] }

// BlockEnd devuelve la última línea (desde 1) del bloque de la tarea de la línea line de la nota: ella y todo lo que le pertenece.
func (s *Storage) BlockEnd(notePath string, line int) (int, error) {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(notePath)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return 0, fmt.Errorf("índice de línea %d fuera de rango", line)
	}
	_, end := taskBlock(lines, line-1)
	return end + 1, nil
}

// shiftBlock corre d columnas el contenido de las líneas de un bloque cuya primera línea tiene ahora su contenido en la columna ci:
// agrega d espacios (d > 0) o quita hasta -d espacios de la sangría (d < 0). Las líneas en blanco no se tocan. Con tabuladores:
//   - si todas las líneas sangradas del bloque usan tab, no se tocan mientras lleguen a ci (un espacio antes de un tab no cambia su ancho y
//     solo ensuciaría el archivo); si no llegan, se reconstruyen con espacios al ancho nuevo;
//   - si el bloque mezcla tabs y espacios, las de tab se corren igual que las de espacios, pero después del tab (el tab se conserva y el
//     ancho cambia en d): así no se igualan niveles y una subtarea no pierde a su hija.
func shiftBlock(lines []string, d, ci int) {
	spaces := false // alguna línea sangrada solo con espacios
	for _, l := range lines {
		if ws := leadingSpace(l); ws != "" && strings.TrimSpace(l) != "" && !strings.Contains(ws, "\t") {
			spaces = true
		}
	}
	for i, l := range lines {
		if d == 0 || strings.TrimSpace(l) == "" {
			continue
		}
		ws := leadingSpace(l)
		if strings.Contains(ws, "\t") {
			switch {
			case spaces && d > 0:
				lines[i] = ws + strings.Repeat(" ", d) + l[len(ws):]
			case spaces:
				trim := len(ws) - len(strings.TrimRight(ws, " ")) // los espacios después del último tab
				lines[i] = ws[:len(ws)-min(trim, -d)] + l[len(ws):]
			default:
				if w := indentWidth(ws); w < ci {
					lines[i] = strings.Repeat(" ", max(0, w+d)) + l[len(ws):]
				}
			}
			continue
		}
		if d > 0 {
			lines[i] = strings.Repeat(" ", d) + l
			continue
		}
		n := 0
		for n < -d && n < len(l) && l[n] == ' ' {
			n++
		}
		lines[i] = l[n:]
	}
}

var orderedMarker = regexp.MustCompile(`^(\s*)(\d{1,9})([.)])`)

// SwapTasks intercambia de lugar las tareas de las líneas lineA y lineB (desde 1) de la misma nota, cada una con todo lo que le
// pertenece (sus subtareas, párrafos y bloques de código): lo que haya entre las dos y el resto del archivo quedan donde estaban, con
// sus terminaciones de línea. Solo las hermanas se intercambian: la misma sangría, el mismo padre y la misma sección (sin un
// encabezado entre las dos), y las dos líneas deben ser tareas. Si son ítems numerados, cada posición conserva su número. Si expected
// no es cero y la nota cambió en disco, no escribe y devuelve ErrNoteChanged. Devuelve la línea nueva de cada una (la tarea de lineA
// queda en newA y la de lineB en newB).
func (s *Storage) SwapTasks(notePath string, lineA, lineB int, expected time.Time) (newA, newB int, err error) {
	notePath, err = s.ResolveNote(notePath)
	if err != nil {
		return 0, 0, err
	}
	swapped := lineA > lineB
	first, second := lineA, lineB
	if swapped {
		first, second = lineB, lineA
	}
	err = rewriteLines(notePath, expected, func(lines []string) ([]string, error) {
		a, b := first-1, second-1
		if first == second || a < 0 || b >= len(lines) {
			return nil, fmt.Errorf("%w: líneas %d y %d", ErrNotSiblings, lineA, lineB)
		}
		for _, i := range []int{a, b} {
			if !taskRegex.MatchString(strings.TrimSuffix(strings.TrimLeft(lines[i], " \t"), "\r")) {
				return nil, fmt.Errorf("%w: la línea %d no es una tarea", ErrNotSiblings, i+1)
			}
		}
		indent := indentWidth(leadingSpace(lines[a]))
		if indent != indentWidth(leadingSpace(lines[b])) {
			return nil, fmt.Errorf("%w: sangrías distintas", ErrNotSiblings)
		}
		_, ea := taskBlock(lines, a)
		_, eb := taskBlock(lines, b)
		if ea >= b { // la segunda es parte del bloque de la primera
			return nil, fmt.Errorf("%w: una cuelga de la otra", ErrNotSiblings)
		}
		// son hermanas solo si son ítems de la misma lista: entre las dos, cada línea es un blanco, un ítem de la misma sangría o algo
		// con más sangría (contenido de otro ítem hermano). Una línea con menos sangría (otro padre), un encabezado, un párrafo o un
		// bloque de código al margen las separan en listas o secciones distintas, y cambiarían de sitio en el documento
		for _, l := range lines[ea+1 : b] {
			t := strings.TrimSuffix(l, "\r")
			if strings.TrimSpace(t) == "" {
				continue
			}
			w := indentWidth(leadingSpace(t))
			if w < indent || (w == indent && !itemMarker.MatchString(t)) {
				return nil, fmt.Errorf("%w: hay otro contenido entre las dos", ErrNotSiblings)
			}
		}
		// cada línea conserva SU terminación (\r o no) en su posición: así un archivo CRLF sin salto final no mezcla terminaciones
		cr := make([]bool, len(lines))
		flat := make([]string, len(lines))
		for i, l := range lines {
			cr[i] = strings.HasSuffix(l, "\r")
			flat[i] = strings.TrimSuffix(l, "\r")
		}
		blockA := append([]string(nil), flat[a:ea+1]...)
		blockB := append([]string(nil), flat[b:eb+1]...)
		// numerados: cada posición conserva su número
		ma, mb := orderedMarker.FindStringSubmatch(blockA[0]), orderedMarker.FindStringSubmatch(blockB[0])
		if ma != nil && mb != nil && ma[3] == mb[3] {
			// el contenido de un ítem numerado empieza tras su número: si cambia el ancho (9. ↔ 10.), el resto del bloque se corre igual
			blockA[0] = ma[1] + mb[2] + ma[3] + blockA[0][len(ma[0]):]
			blockB[0] = mb[1] + ma[2] + mb[3] + blockB[0][len(mb[0]):]
			shiftBlock(blockA[1:], len(mb[2])-len(ma[2]), contentIndent(blockA[0]))
			shiftBlock(blockB[1:], len(ma[2])-len(mb[2]), contentIndent(blockB[0]))
		}
		out := make([]string, 0, len(lines))
		out = append(out, flat[:a]...)
		out = append(out, blockB...)
		out = append(out, flat[ea+1:b]...)
		out = append(out, blockA...)
		out = append(out, flat[eb+1:]...)
		for i := range out {
			if cr[i] {
				out[i] += "\r"
			}
		}
		newFirst, newSecond := first+(eb-ea), first // la primera baja detrás de la segunda; la segunda sube al lugar de la primera
		if swapped {
			newA, newB = newSecond, newFirst
		} else {
			newA, newB = newFirst, newSecond
		}
		return out, nil
	})
	if err != nil {
		return 0, 0, err
	}
	return newA, newB, nil
}
