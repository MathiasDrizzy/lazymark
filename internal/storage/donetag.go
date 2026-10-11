package storage

import (
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// tagSpans returns the spans of the whole-token occurrences of #tag in text (case-insensitive). A token starts the text or
// follows a space or tab, and runs over letters, digits, "_", "-" and "/", so "#archived-later" and "#archived/sub" are not "#archived".
func tagSpans(text, tag string) [][2]int {
	want := "#" + strings.ToLower(tag)
	var spans [][2]int
	for _, loc := range tokenRe.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && text[loc[0]-1] != ' ' && text[loc[0]-1] != '\t' {
			continue
		}
		if strings.ToLower(text[loc[0]:loc[1]]) == want {
			spans = append(spans, [2]int{loc[0], loc[1]})
		}
	}
	return spans
}

// HasTag reports whether text carries #tag as a whole tag (tag is given without the "#").
func HasTag(text, tag string) bool { return len(tagSpans(text, tag)) > 0 }

// RewriteToggleTag adds #tag at the end of a task line, or removes every #tag from it if it is already there. It returns the
// new line and whether the tag was added. Nothing else on the line changes.
func RewriteToggleTag(line, tag string) (string, bool) {
	spans := tagSpans(line, tag)
	if len(spans) == 0 {
		crlf := strings.HasSuffix(line, "\r")
		out := strings.TrimRight(line, " \t\r") + " #" + tag
		if crlf {
			out += "\r"
		}
		return out, true
	}
	for i := len(spans) - 1; i >= 0; i-- {
		line = removeSpan(line, spans[i][0], spans[i][1])
	}
	return line, false
}

// ToggleTaskTag adds or removes #tag on the task at the given line, rewriting only that line. If expected is not zero and the note
// changed on disk since it was read, it writes nothing and returns ErrNoteChanged.
func (s *Storage) ToggleTaskTag(notePath string, line int, tag string, expected time.Time) (bool, error) {
	notePath, err := s.ResolveNote(notePath)
	if err != nil {
		return false, err
	}
	var added bool
	err = rewriteLine(notePath, line, expected, func(l string) (string, error) {
		if !taskRegex.MatchString(strings.TrimLeft(l, " \t")) {
			return "", i18n.Errorf("la línea %d no es una tarea", "line %d is not a task", line)
		}
		out, a := RewriteToggleTag(l, tag)
		added = a
		return out, nil
	})
	return added && err == nil, err
}
