package storage

import (
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Las fechas de una tarea van al final de su línea, en uno de los dos formatos de Obsidian Tasks (que son markdown inocuo: GFM los muestra como
// texto): con emojis (inicio `🛫 2026-05-01`, vencimiento `📅 2026-05-10`, completada `✅ 2026-05-09`, programada `⏳ …`, creada `➕ …`) o en el
// formato Dataview (`[start:: 2026-05-01]`, `[due:: …]`, `[completion:: …]`, `[scheduled:: …]`, `[created:: …]`, también con paréntesis). Se leen
// siempre los dos; al escribir se usa el formato de la configuración (date_format), o el que esa línea ya tenía.

// DateField es uno de los campos de fecha.
type DateField int

const (
	DateStart DateField = iota
	DateDue
	DateDone
	DateScheduled
	DateCreated
	numDateFields
)

// DateFormat es el formato en que se escribe una fecha.
type DateFormat int

const (
	FormatEmoji DateFormat = iota
	FormatDataview
)

var (
	dateEmoji = [numDateFields]string{"🛫", "📅", "✅", "⏳", "➕"}
	dateKey   = [numDateFields]string{"start", "due", "completion", "scheduled", "created"}
)

// Emoji es el emoji de Obsidian Tasks que abre el campo.
func (f DateField) Emoji() string { return dateEmoji[f] }

// Key es el nombre del campo en el formato Dataview (`[due:: …]`).
func (f DateField) Key() string { return dateKey[f] }

// emojiDateRe encuentra un emoji de fecha con su fecha (el selector de variación U+FE0F y los espacios son opcionales); la fecha debe terminar ahí
// (\b: no sigue otro dígito). dataviewDateRe, un campo Dataview entre corchetes o paréntesis, con espacios opcionales alrededor de "::" y dentro.
var (
	emojiDateRe    = regexp.MustCompile(`(🛫|📅|✅|⏳|➕)\x{FE0F}?[ \t]*(\d{4}-\d{2}-\d{2})\b`)
	dataviewDateRe = regexp.MustCompile(`\[[ \t]*(start|due|completion|scheduled|created)[ \t]*::[ \t]*(\d{4}-\d{2}-\d{2})[ \t]*\]|\([ \t]*(start|due|completion|scheduled|created)[ \t]*::[ \t]*(\d{4}-\d{2}-\d{2})[ \t]*\)`)
)

// ValidDate indica si s es una fecha real del calendario con la forma AAAA-MM-DD.
func ValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10
}

// Dates son las fechas de una tarea ("" si no tiene ese campo).
type Dates struct{ Start, Due, Done, Scheduled, Created string }

type dateHit struct {
	field      DateField
	format     DateFormat
	paren      bool // Dataview entre paréntesis en vez de corchetes
	start, end int  // el span del marcador (emoji o campo Dataview) y su fecha en el texto
	date       string
	valid      bool // la fecha existe en el calendario
}

// render escribe un marcador del campo f con la fecha value en el formato fm (con los paréntesis de h si h es Dataview con paréntesis).
func renderMarker(f DateField, value string, fm DateFormat, paren bool) string {
	if fm == FormatDataview {
		if paren {
			return "(" + f.Key() + ":: " + value + ")"
		}
		return "[" + f.Key() + ":: " + value + "]"
	}
	return f.Emoji() + " " + value
}

// scanDates devuelve los campos de fecha válidos de text, en orden de aparición. Una fecha inválida no es un campo.
func scanDates(text string) []dateHit {
	var hits []dateHit
	for _, h := range scanMarkers(text) {
		if h.valid {
			hits = append(hits, h)
		}
	}
	return hits
}

// scanMarkers devuelve todos los marcadores de fecha de text (un emoji de fecha o un campo Dataview seguido de algo con forma de fecha
// AAAA-MM-DD), válidos o no, en orden de aparición. Al escribir, un marcador inválido del mismo campo cuenta: se reemplaza o se quita, para que
// nunca queden dos del mismo campo.
func scanMarkers(text string) []dateHit {
	var hits []dateHit
	for _, m := range emojiDateRe.FindAllStringSubmatchIndex(text, -1) {
		date := text[m[4]:m[5]]
		f := DateStart
		for i, e := range dateEmoji {
			if text[m[2]:m[3]] == e {
				f = DateField(i)
			}
		}
		hits = append(hits, dateHit{field: f, format: FormatEmoji, start: m[0], end: m[1], date: date, valid: ValidDate(date)})
	}
	for _, m := range dataviewDateRe.FindAllStringSubmatchIndex(text, -1) {
		k, d, paren := 2, 4, false // [clave:: fecha] o (clave:: fecha)
		if m[2] >= 0 {
			k, d = 2, 4
			// `[[due:: 2026-05-10]]` es un wikilink, no un campo
			if (m[0] > 0 && text[m[0]-1] == '[') || (m[1] < len(text) && text[m[1]] == ']') {
				continue
			}
		} else {
			k, d, paren = 6, 8, true
		}
		var f DateField
		for i, key := range dateKey {
			if text[m[k]:m[k+1]] == key {
				f = DateField(i)
			}
		}
		date := text[m[d]:m[d+1]]
		hits = append(hits, dateHit{field: f, format: FormatDataview, paren: paren, start: m[0], end: m[1], date: date, valid: ValidDate(date)})
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].start < hits[j].start })
	return hits
}

// ParseDates lee las fechas de un texto de tarea: de cada campo vale la primera fecha válida.
func ParseDates(text string) Dates {
	var d Dates
	for _, h := range scanDates(text) {
		switch {
		case h.field == DateStart && d.Start == "":
			d.Start = h.date
		case h.field == DateDue && d.Due == "":
			d.Due = h.date
		case h.field == DateDone && d.Done == "":
			d.Done = h.date
		case h.field == DateScheduled && d.Scheduled == "":
			d.Scheduled = h.date
		case h.field == DateCreated && d.Created == "":
			d.Created = h.date
		}
	}
	return d
}

// Overdue indica si la tarea está vencida: no está hecha y su vencimiento es anterior a today (AAAA-MM-DD).
func Overdue(done bool, due, today string) bool {
	return !done && due != "" && due < today // las fechas AAAA-MM-DD se ordenan como texto
}

// Today es la fecha de hoy (AAAA-MM-DD); los tests la reemplazan.
var Today = func() string { return time.Now().Format("2006-01-02") }

// checkboxOnly reconoce una línea que termina en la casilla ("- [ ]"), sin texto ni espacio detrás.
var checkboxOnly = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+\[[ xX]\]$`)

// setDate es setDateIn en el formato de emojis.
func setDate(line string, f DateField, value string) string {
	return setDateIn(line, f, value, FormatEmoji)
}

// setDateIn devuelve la línea con el campo f puesto en value (AAAA-MM-DD) o quitado (value ""). Con un valor, reemplaza en su sitio el primer
// marcador del campo (aunque su fecha sea inválida), en el formato que ese marcador ya tenía, y quita los demás: nunca quedan dos del mismo
// campo; si no hay ninguno, lo agrega al final (antes de la fecha de completada, si la hay, y antes del espacio y el \r finales), en el formato de
// los marcadores que la línea ya tenga o, si no tiene ninguno, en newFmt. Con value "" quita todos los marcadores del campo. Al quitar, los espacios
// alrededor se normalizan: el texto no queda pegado a otro marcador. No toca nada más de la línea.
func setDateIn(line string, f DateField, value string, newFmt DateFormat) string {
	if value != "" && !ValidDate(value) {
		panic("setDate: fecha inválida " + value) // los llamadores validan antes
	}
	cr := ""
	body := line
	if strings.HasSuffix(body, "\r") {
		cr, body = "\r", strings.TrimSuffix(body, "\r")
	}
	var mine []dateHit
	all := scanMarkers(body)
	for _, h := range all {
		if h.field == f {
			mine = append(mine, h)
		}
	}
	fm, paren := newFmt, false
	switch {
	case len(mine) > 0: // se reemplaza en su sitio y en su formato
		fm, paren = mine[0].format, mine[0].paren
	case len(all) > 0: // un campo nuevo en una línea que ya tiene fechas: el mismo formato que ellas
		fm, paren = all[0].format, all[0].paren
	}
	field := renderMarker(f, value, fm, paren)
	if len(mine) > 0 {
		firstToRemove := 1 // con un valor se reemplaza el primero y se quitan los demás
		if value == "" {
			firstToRemove = 0 // sin valor se quitan todos
		}
		for i := len(mine) - 1; i >= firstToRemove; i-- { // de atrás hacia adelante: los índices anteriores siguen valiendo
			body = removeSpan(body, mine[i].start, mine[i].end)
		}
		if value != "" {
			after := body[mine[0].end:]
			if after != "" && after[0] != ' ' && after[0] != '\t' { // no se pega a lo que sigue (otro emoji, texto)
				after = " " + after
			}
			body = body[:mine[0].start] + field + after
		} else if checkboxOnly.MatchString(body) {
			body += " " // "- [ ]" sin nada detrás deja de ser una tarea: se conserva el espacio de la casilla
		}
		return body + cr
	}
	if value == "" {
		return line
	}
	trimmed := strings.TrimRight(body, " \t")
	tail := body[len(trimmed):]
	at := len(trimmed)
	if f != DateDone { // el marcador de completada va siempre al final (también si su fecha es inválida)
		for _, h := range all {
			if h.field == DateDone {
				at = h.start
				for at > 0 && (trimmed[at-1] == ' ' || trimmed[at-1] == '\t') {
					at--
				}
				break
			}
		}
	}
	rest := trimmed[at:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		rest = " " + rest // lo que sigue (el ✅ pegado a un texto) no queda pegado al campo nuevo
	}
	return trimmed[:at] + " " + field + rest + tail + cr
}

// SetTaskDate pone (value "AAAA-MM-DD") o quita (value "") el campo f de la tarea de la línea line de la nota, reescribiendo
// solo esa línea y sin pisar una nota que cambió en disco (expected, como en MoveTask). El campo "completada" lo maneja el
// movimiento a la columna de hecho: aquí solo se permiten el inicio y el vencimiento.
func (s *Storage) SetTaskDate(notePath string, line int, f DateField, value string, expected time.Time) error {
	if f == DateDone || f == DateCreated {
		return fmt.Errorf("la fecha de completada y la de creación no se editan a mano")
	}
	if value != "" && !ValidDate(value) {
		return fmt.Errorf("%q no es una fecha válida (AAAA-MM-DD)", value)
	}
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	return rewriteLine(notePath, line, expected, func(l string) (string, error) {
		if !toggleTaskRegex.MatchString(l) {
			return "", fmt.Errorf("la línea %d no es una tarea válida de markdown", line)
		}
		return setDateIn(l, f, value, s.WriteDateFormat()), nil
	})
}

// SetTaskDates pone, cambia o quita el inicio y el vencimiento de la tarea de la línea line en una sola escritura de esa línea (el popup de
// fechas): nil deja el campo como está, "" lo quita y "AAAA-MM-DD" lo pone. Las dos se validan antes de escribir y, como en SetTaskDate,
// una nota cambiada en disco (expected) no se pisa.
func (s *Storage) SetTaskDates(notePath string, line int, start, due *string, expected time.Time) error {
	for _, v := range []*string{start, due} {
		if v != nil && *v != "" && !ValidDate(*v) {
			return fmt.Errorf("%q no es una fecha válida (AAAA-MM-DD)", *v)
		}
	}
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return err
	}
	return rewriteLine(notePath, line, expected, func(l string) (string, error) {
		if !toggleTaskRegex.MatchString(l) {
			return "", fmt.Errorf("la línea %d no es una tarea válida de markdown", line)
		}
		fm := s.WriteDateFormat()
		if start != nil {
			l = setDateIn(l, DateStart, *start, fm)
		}
		if due != nil {
			l = setDateIn(l, DateDue, *due, fm)
		}
		return l, nil
	})
}

// WriteDateFormat es el formato en que se escribe una fecha nueva en una línea que aún no tiene fechas: el de la configuración (DateFormatPref) o,
// por defecto, Dataview; salvo que el vault ya tenga tareas con emojis y ninguna con Dataview, en cuyo caso es emoji (y queda pendiente un aviso, una
// sola vez). Se calcula la primera vez que hace falta y se recuerda (ResetDateFormat lo olvida).
func (s *Storage) WriteDateFormat() DateFormat {
	switch s.DateFormatPref {
	case "emoji":
		return FormatEmoji
	case "dataview":
		return FormatDataview
	}
	s.fmtMu.Lock()
	defer s.fmtMu.Unlock()
	if !s.fmtKnown {
		emoji, dv := s.countDateFormats()
		s.fmtVault, s.fmtExcept = FormatDataview, false
		if emoji > 0 && dv == 0 {
			s.fmtVault, s.fmtExcept = FormatEmoji, true
		}
		s.fmtKnown = true
	}
	if s.fmtExcept && !s.DateNoticeSeen && s.notice == "" {
		s.notice = i18n.T("Este vault usa emojis en las fechas: se escriben así (cambiar: Ajustes → Formato de fechas o date_format; migrar: lazymark dates migrate).",
			"This vault uses emoji dates: they are written that way (change: Settings → Date format or date_format; migrate: lazymark dates migrate).")
	}
	return s.fmtVault
}

// ResetDateFormat hace olvidar el formato detectado del vault (cambió la carpeta o se migraron notas).
func (s *Storage) ResetDateFormat() {
	s.fmtMu.Lock()
	s.fmtKnown = false
	s.fmtMu.Unlock()
}

// TakeDateFormatNotice devuelve el aviso del formato de fechas (una sola vez: vacío después) y lo marca como visto; el llamador lo muestra y guarda
// DateNoticeSeen en la configuración.
func (s *Storage) TakeDateFormatNotice() string {
	s.fmtMu.Lock()
	defer s.fmtMu.Unlock()
	n := s.notice
	if n != "" {
		s.notice, s.DateNoticeSeen = "", true
	}
	return n
}

// countDateFormats cuenta las tareas del vault con fechas en emoji y las que las tienen en Dataview (una tarea con ambos formatos cuenta en los dos).
func (s *Storage) countDateFormats() (emoji, dataview int) {
	notes, err := s.ListNotes()
	if err != nil {
		return 0, 0
	}
	for _, n := range notes {
		for _, t := range n.Tasks {
			e, d := false, false
			for _, h := range scanMarkers(t.Text) {
				e, d = e || h.format == FormatEmoji, d || h.format == FormatDataview
			}
			if e {
				emoji++
			}
			if d {
				dataview++
			}
		}
	}
	return emoji, dataview
}

// DateChange es un cambio de MigrateDates: una línea de tarea de una nota, antes y después.
type DateChange struct {
	Path, Rel     string
	Line          int
	Before, After string
}

// MigrateDates pasa las fechas de todas las tareas de las notas al formato to: cada marcador válido se reescribe en ese formato (las fechas inválidas
// y todo lo que no es una línea de tarea —párrafos, bloques de código— quedan intactos). Con dryRun solo devuelve lo que cambiaría. Es idempotente:
// una segunda corrida no devuelve cambios. Cada nota se escribe de forma atómica, y solo si no cambió mientras tanto (ErrNoteChanged).
func (s *Storage) MigrateDates(to DateFormat, dryRun bool) ([]DateChange, error) {
	notes, err := s.ListNotes()
	if err != nil {
		return nil, err
	}
	var out []DateChange
	for _, n := range notes {
		if n.TooLarge || len(n.Tasks) == 0 {
			continue
		}
		taskLine := map[int]bool{}
		for _, t := range n.Tasks {
			taskLine[t.Line] = true
		}
		var changes []DateChange
		lines := strings.Split(n.Content, "\n")
		for i, l := range lines {
			if !taskLine[i+1] {
				continue
			}
			if after := convertDates(l, to); after != l {
				changes = append(changes, DateChange{Path: n.Path, Rel: s.relPath(n.Path), Line: i + 1, Before: strings.TrimSuffix(l, "\r"), After: strings.TrimSuffix(after, "\r")})
			}
		}
		if len(changes) == 0 {
			continue
		}
		out = append(out, changes...)
		if dryRun {
			continue
		}
		err := rewriteLines(n.Path, n.ModTime, func(ls []string) ([]string, error) {
			for _, c := range changes {
				if c.Line-1 >= len(ls) {
					return nil, ErrNoteChanged
				}
				cr := ""
				if strings.HasSuffix(ls[c.Line-1], "\r") {
					cr = "\r"
				}
				if strings.TrimSuffix(ls[c.Line-1], "\r") != c.Before {
					return nil, ErrNoteChanged
				}
				ls[c.Line-1] = c.After + cr
			}
			return ls, nil
		})
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// convertDates reescribe cada marcador de fecha válido de la línea en el formato to (los corchetes de Dataview; sin paréntesis), sin tocar nada más.
func convertDates(line string, to DateFormat) string {
	hits := scanDates(line)
	for i := len(hits) - 1; i >= 0; i-- {
		h := hits[i]
		marker := renderMarker(h.field, h.date, to, false)
		if r, _ := utf8.DecodeRuneInString(line[h.end:]); h.end < len(line) && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			marker += " " // un marcador pegado a texto no se funde con él (la fecha dejaría de leerse)
		}
		line = line[:h.start] + marker + line[h.end:]
	}
	return line
}
