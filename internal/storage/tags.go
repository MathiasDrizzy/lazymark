package storage

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var tagTokenRe = regexp.MustCompile(`#[\p{L}\p{N}_-]+(?:/[\p{L}\p{N}_/-]*)?`)
var tagURLRe = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s<>]+`)

// extractTags keeps category names (the first segment of nested tags), lowercased and sorted.
func (s *Storage) extractTags(content string) []string {
	text := tagProse(content)
	mask := tagCodeMask(text)
	for _, re := range []*regexp.Regexp{wikilinkRe, tagURLRe} {
		for _, span := range re.FindAllStringIndex(text, -1) {
			maskTagSpan(mask, span[0], span[1])
		}
	}
	// Destinations can contain whitespace, escaped parentheses and nested parentheses.
	for i := 0; i+1 < len(text); i++ {
		if text[i] != ']' || text[i+1] != '(' || mask[i] {
			continue
		}
		depth := 1
		j := i + 2
		for ; j < len(text) && depth > 0; j++ {
			if text[j] == '\\' && j+1 < len(text) {
				j++
				continue
			}
			switch text[j] {
			case '(':
				depth++
			case ')':
				depth--
			}
		}
		if depth == 0 {
			maskTagSpan(mask, i, j)
		}
		i = j - 1
	}
	tags := map[string]bool{}
	for _, span := range tagTokenRe.FindAllStringIndex(text, -1) {
		start := span[0]
		if mask[start] {
			continue
		}
		if start > 0 {
			prev, _ := utf8.DecodeLastRuneInString(text[:start])
			if !unicode.IsSpace(prev) {
				continue
			}
		}
		tag := strings.ToLower(text[start+1 : span[1]])
		root, _, _ := strings.Cut(tag, "/")
		if strings.HasPrefix("#"+tag, kanbanPrefix()) {
			continue
		}
		if strings.IndexFunc(root, func(r rune) bool { return !unicode.IsNumber(r) }) < 0 {
			continue
		}
		tags[root] = true
	}
	var result []string
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// tagProse skips frontmatter and uses the same fence and list rules as extractTasks.
func tagProse(content string) string {
	frontmatter := false
	var (
		fence       string // el vallado de código abierto (``` o ~~~, con su largo), o ""
		fenceIndent int    // la sangría con que se abrió: una línea con menos cierra el ítem que lo contenía, y con él el vallado
		fenceInList bool   // se abrió dentro de una lista: ahí un vallado puede llevar 4 o más espacios de sangría
		inList      bool   // estamos dentro de una lista: ahí una sangría de 4 o más es una sublista y no un bloque de código
	)
	lines := strings.Split(content, "\n")
	for i, raw := range lines {
		lines[i] = ""
		line := strings.TrimSuffix(raw, "\r")
		if i == 0 && line == "---" {
			frontmatter = true
			continue
		}
		if frontmatter {
			if line == "---" || line == "..." {
				frontmatter = false
			}
			continue
		}
		body := strings.TrimLeft(line, " \t")
		indent := indentWidth(line[:len(line)-len(body)])
		if fence != "" && body != "" && indent < fenceIndent {
			fence = "" // CommonMark: un vallado sin cerrar termina con el ítem de lista que lo contiene
		}
		if f := fenceMarker(body); f != "" && (indent < 4 || inList || fence != "") {
			switch {
			case fence == "":
				fence, fenceIndent, fenceInList = f, indent, inList
				if indent == 0 {
					inList = false // un vallado al margen corta la lista
				}
			case f[0] == fence[0] && len(f) >= len(fence) && strings.TrimSpace(body)[len(f):] == "" && (indent < 4 || fenceInList):
				fence = ""
			}
			continue
		}
		if fence != "" { // Code blocks do not contribute tags.
			continue
		}
		if body == "" { // las líneas en blanco no cortan una lista
			continue
		}
		item := listItemRegex.MatchString(body)
		switch {
		case item && indent < 4:
			inList = true
		case !item && indent == 0:
			inList = false // un párrafo pegado al margen termina la lista
		}

		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func maskTagSpan(mask []bool, start, end int) {
	for i := start; i < end; i++ {
		mask[i] = true
	}
}

// Index runs once so unmatched backticks do not cause repeated scans of long lines.
func tagCodeMask(text string) []bool {
	type run struct{ start, end, next int }
	var runs []run
	last := map[int]int{}
	for i := 0; i < len(text); {
		if text[i] != '`' {
			i++
			continue
		}
		end := i + 1
		for end < len(text) && text[end] == '`' {
			end++
		}
		if previous, ok := last[end-i]; ok {
			runs[previous].next = len(runs)
		}
		last[end-i] = len(runs)
		runs = append(runs, run{i, end, -1})
		i = end
	}
	mask := make([]bool, len(text))
	for i := 0; i < len(runs); i++ {
		close := runs[i].next
		if close < 0 {
			continue
		}
		maskTagSpan(mask, runs[i].start, runs[close].end)
		i = close
	}
	return mask
}
