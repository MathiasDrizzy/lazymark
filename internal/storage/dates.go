package storage

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Las fechas de una tarea van en el formato de Obsidian Tasks, que es markdown inocuo (GFM las muestra como texto):
// inicio `🛫 2026-05-01`, vencimiento `📅 2026-05-10` y completada `✅ 2026-05-09`, al final de la línea.

// DateField es uno de los tres campos de fecha.
type DateField int

const (
	DateStart DateField = iota
	DateDue
	DateDone
)

var dateEmoji = [3]string{"🛫", "📅", "✅"}

// Name es el nombre del campo en la línea de comandos y en el JSON: start, due, done.
func (f DateField) Name() string { return [3]string{"start", "due", "done"}[f] }

// Emoji es el emoji que abre el campo.
func (f DateField) Emoji() string { return dateEmoji[f] }

// dateRe encuentra un emoji de fecha con su fecha (el selector de variación U+FE0F y los espacios son opcionales); la
// fecha debe terminar ahí (\b: no sigue otro dígito).
var dateRe = regexp.MustCompile(`(🛫|📅|✅)\x{FE0F}?[ \t]*(\d{4}-\d{2}-\d{2})\b`)

// ValidDate indica si s es una fecha real del calendario con la forma AAAA-MM-DD.
func ValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10
}

// Dates son las fechas de una tarea ("" si no tiene ese campo).
type Dates struct{ Start, Due, Done string }

// Get devuelve la fecha del campo f.
func (d Dates) Get(f DateField) string { return [3]string{d.Start, d.Due, d.Done}[f] }

type dateHit struct {
	field      DateField
	start, end int // el span del emoji y la fecha en el texto
	date       string
}

// scanDates devuelve los campos de fecha válidos de text, en orden de aparición. Una fecha inválida no es un campo.
func scanDates(text string) []dateHit {
	var hits []dateHit
	for _, m := range dateRe.FindAllStringSubmatchIndex(text, -1) {
		date := text[m[4]:m[5]]
		if !ValidDate(date) {
			continue
		}
		var f DateField
		switch text[m[2]:m[3]] {
		case dateEmoji[DateStart]:
			f = DateStart
		case dateEmoji[DateDue]:
			f = DateDue
		default:
			f = DateDone
		}
		hits = append(hits, dateHit{field: f, start: m[0], end: m[1], date: date})
	}
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

// setDate devuelve la línea con el campo f puesto en value (AAAA-MM-DD) o quitado (value ""): reemplaza en su sitio la
// primera fecha válida del campo; si no hay, la agrega al final (antes de la fecha de completada, si la hay, y antes
// del espacio y el \r finales). No toca nada más de la línea.
func setDate(line string, f DateField, value string) string {
	if value != "" && !ValidDate(value) {
		panic("setDate: fecha inválida " + value) // los llamadores validan antes
	}
	cr := ""
	body := line
	if strings.HasSuffix(body, "\r") {
		cr, body = "\r", strings.TrimSuffix(body, "\r")
	}
	hits := scanDates(body)
	first := -1
	for i, h := range hits {
		if h.field == f {
			first = i
			break
		}
	}
	field := f.Emoji() + " " + value
	switch {
	case first >= 0 && value == "":
		h := hits[first]
		start := h.start
		for start > 0 && (body[start-1] == ' ' || body[start-1] == '\t') {
			start--
		}
		body = body[:start] + body[h.end:]
	case first >= 0:
		h := hits[first]
		body = body[:h.start] + field + body[h.end:]
	case value != "":
		trimmed := strings.TrimRight(body, " \t")
		tail := body[len(trimmed):]
		at := len(trimmed)
		if f != DateDone { // el emoji de completada va siempre al final
			for _, h := range hits {
				if h.field == DateDone {
					at = h.start
					for at > 0 && (trimmed[at-1] == ' ' || trimmed[at-1] == '\t') {
						at--
					}
					break
				}
			}
		}
		body = trimmed[:at] + " " + field + trimmed[at:] + tail
	}
	return body + cr
}

// SetTaskDate pone (value "AAAA-MM-DD") o quita (value "") el campo f de la tarea de la línea line de la nota, reescribiendo
// solo esa línea y sin pisar una nota que cambió en disco (expected, como en MoveTask). El campo "completada" lo maneja el
// movimiento a la columna de hecho: aquí solo se permiten el inicio y el vencimiento.
func (s *Storage) SetTaskDate(notePath string, line int, f DateField, value string, expected time.Time) error {
	if f == DateDone {
		return fmt.Errorf("la fecha de completada se pone sola al marcar la tarea como hecha")
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
		return setDate(l, f, value), nil
	})
}
