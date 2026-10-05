package storage

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var liRe = regexp.MustCompile(`(?s)<li>.*?</li>`)

// itemsHTML devuelve, ordenado, el HTML de cada ítem de lista de src según goldmark (el analizador de la vista previa): una huella de
// "qué contiene cada ítem". Intercambiar dos ítems hermanos no debe cambiar este conjunto.
func itemsHTML(t *testing.T, src string) []string {
	t.Helper()
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(extension.TaskList)).Convert([]byte(src), &buf); err != nil {
		t.Fatal(err)
	}
	// los <li> pueden anidarse: se toma cada uno desde su apertura hasta su cierre equilibrado
	html := buf.String()
	var out []string
	for i := 0; i < len(html); {
		j := strings.Index(html[i:], "<li")
		if j < 0 {
			break
		}
		start, depth, k := i+j, 0, i+j
		for k < len(html) {
			o, c := strings.Index(html[k:], "<li"), strings.Index(html[k:], "</li>")
			if c < 0 {
				break
			}
			if o >= 0 && o < c {
				depth++
				k += o + 3
				continue
			}
			depth--
			k += c + 5
			if depth == 0 {
				break
			}
		}
		out = append(out, html[start:k])
		i = start + 3
	}
	sort.Strings(out)
	return out
}

// TestSwapTasksKeepsEveryItemWhole (C.4b R1): al intercambiar dos tareas hermanas, cada ítem conserva TODO lo que le pertenece según
// CommonMark: bloques de código con líneas en blanco, párrafos de continuación tras una línea en blanco, sublistas. El oráculo es
// goldmark: el conjunto de ítems (su HTML) es el mismo antes y después, y el archivo tiene las mismas líneas.
func TestSwapTasksKeepsEveryItemWhole(t *testing.T) {
	cases := map[string]struct {
		doc          string
		lineA, lineB int
	}{
		"bloque de código con línea en blanco": {"- [ ] Tarea A\n  ```go\n  x := 1\n\n  y := 2\n  ```\n- [ ] Tarea B\n", 1, 7},
		"párrafo de continuación tras blanco":  {"- [ ] Tarea A\n\n  continuación de A\n- [ ] Tarea B\n", 1, 4},
		"sublista y párrafo":                   {"- [ ] A\n  - [ ] a1\n\n    más de a1\n  - [ ] a2\n\n  párrafo de A\n- [ ] B\n  - [ ] b1\n", 1, 8},
		"blancos entre medio no se mueven":     {"- [ ] A\n  texto de A\n\n\n- [ ] B\n- [ ] C\n", 1, 5},
		"numeradas":                            {"1. [ ] uno\n   detalle\n2. [ ] dos\n3. [ ] tres\n", 1, 3},
		"vallado con tilde y blancos":          {"- [ ] A\n  ~~~\n  a\n\n\n  b\n  ~~~\n- [ ] B\n  ```\n  c\n  ```\n", 1, 8},
		"con tabuladores":                      {"- [ ] A\n\t- [ ] a1\n\n\tdetalle\n- [ ] B\n", 1, 5},
		"CRLF":                                 {"- [ ] A\r\n\r\n  cont\r\n- [ ] B\r\n", 1, 4},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			s, p := kanbanNote(t, c.doc)
			before := itemsHTML(t, c.doc)
			if _, _, err := s.SwapTasks(p, c.lineA, c.lineB, time.Time{}); err != nil {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(p)
			after := string(b)
			if got := itemsHTML(t, after); strings.Join(got, "\n") != strings.Join(before, "\n") {
				t.Errorf("los ítems cambiaron:\nantes  %q\ndespués %q\nnota:\n%q", before, got, after)
			}
			if strings.Count(after, "\n") != strings.Count(c.doc, "\n") || len(after) != len(c.doc) {
				t.Errorf("cambió la cantidad de líneas o de bytes:\n%q", after)
			}
			if first := strings.SplitAfter(after, "\n")[0]; !strings.HasSuffix(first, textOf(c.doc, c.lineB)) {
				t.Errorf("la segunda tarea debe quedar arriba:\n%q", after)
			}
		})
	}
}

// textOf devuelve lo que sigue a la casilla en la línea n de doc, con su terminación de línea (para ver qué quedó arriba).
func textOf(doc string, n int) string {
	l := strings.SplitAfter(doc, "\n")[n-1]
	return l[strings.Index(l, "]")+1:]
}

// TestSwapTasksSecondOpinionItems (C.4b R3): los ítems restantes de la segunda opinión del apoyo, cada uno con su test.
func TestSwapTasksSecondOpinionItems(t *testing.T) {
	swap := func(t *testing.T, doc string, a, b int) (string, error) {
		t.Helper()
		s, p := kanbanNote(t, doc)
		_, _, err := s.SwapTasks(p, a, b, time.Time{})
		got, _ := os.ReadFile(p)
		return string(got), err
	}
	t.Run("los casos de la auditoría", func(t *testing.T) {
		// un vallado al margen separa las dos tareas en listas distintas (CommonMark): no son hermanas, no se tocan
		doc := "- [ ] Tarea A\n```go\nx := 1\n\ny := 2\n```\n- [ ] Tarea B\n"
		got, err := swap(t, doc, 1, 7)
		if !errors.Is(err, ErrNotSiblings) || got != doc {
			t.Errorf("vallado al margen entre las dos: %v\n%q", err, got)
		}
		got, err = swap(t, "- [ ] Tarea A\n  ```go\n  x := 1\n\n  y := 2\n  ```\n- [ ] Tarea B\n", 1, 7)
		if err != nil || got != "- [ ] Tarea B\n- [ ] Tarea A\n  ```go\n  x := 1\n\n  y := 2\n  ```\n" {
			t.Errorf("bloque de código dentro de la tarea: %v\n%q", err, got)
		}
		got, err = swap(t, "- [ ] Tarea A\n\n  continuación de A\n- [ ] Tarea B\n", 1, 4)
		if err != nil || got != "- [ ] Tarea B\n- [ ] Tarea A\n\n  continuación de A\n" {
			t.Errorf("párrafo de continuación: %v\n%q", err, got)
		}
	})
	t.Run("numeradas: cada posición conserva su número", func(t *testing.T) {
		got, err := swap(t, "1. [ ] uno\n2. [ ] dos\n3. [ ] tres\n", 1, 3)
		if err != nil || got != "1. [ ] tres\n2. [ ] dos\n3. [ ] uno\n" {
			t.Errorf("%v\n%q", err, got)
		}
		got, err = swap(t, "1) [ ] a\n   detalle\n2) [ ] b\n", 1, 3)
		if err != nil || got != "1) [ ] b\n2) [ ] a\n   detalle\n" {
			t.Errorf("con paréntesis y continuación: %v\n%q", err, got)
		}
	})
	t.Run("indent 0 bajo encabezados distintos: no son hermanas", func(t *testing.T) {
		doc := "## Lunes\n- [ ] a\n\n## Martes\n- [ ] b\n"
		got, err := swap(t, doc, 2, 5)
		if !errors.Is(err, ErrNotSiblings) || got != doc {
			t.Errorf("a y b están en secciones distintas: %v\n%q", err, got)
		}
		doc = "## Lunes\n- [ ] a\n- [ ] x\n\n- [ ] b\n"
		if got, err := swap(t, doc, 2, 5); err != nil || got != "## Lunes\n- [ ] b\n- [ ] x\n\n- [ ] a\n" {
			t.Errorf("misma sección: %v\n%q", err, got)
		}
	})
	t.Run("un párrafo al margen separa las listas", func(t *testing.T) {
		doc := "- [ ] Tarea 1\n\nPárrafo intermedio no sangrado.\n\n- [ ] Tarea 2\n"
		if got, err := swap(t, doc, 1, 5); !errors.Is(err, ErrNotSiblings) || got != doc {
			t.Errorf("%v\n%q", err, got)
		}
	})
	t.Run("numeradas con 9 y 10: el contenido se corre con el número", func(t *testing.T) {
		got, err := swap(t, "9. [ ] Nueve\n   - [ ] sub\n10. [ ] Diez\n    - [ ] sub10\n", 1, 3)
		if err != nil || got != "9. [ ] Diez\n   - [ ] sub10\n10. [ ] Nueve\n    - [ ] sub\n" {
			t.Errorf("%v\n%q", err, got)
		}
		doc := "9. [ ] Nueve\n   - [ ] sub\n10. [ ] Diez\n    - [ ] sub10\n"
		s, p := kanbanNote(t, doc)
		s.SwapTasks(p, 1, 3, time.Time{})
		b, _ := os.ReadFile(p)
		if got, want := itemsHTML(t, string(b)), itemsHTML(t, doc); len(got) != len(want) {
			t.Errorf("la cantidad de ítems cambió: %d → %d\n%q", len(want), len(got), b)
		}
	})
	t.Run("tab tras la viñeta", func(t *testing.T) {
		// "-\t[ ] a": el contenido empieza en la columna 4; una continuación con 4 espacios es suya
		got, err := swap(t, "-\t[ ] a\n    cont\n- [ ] b\n", 1, 3)
		if err != nil || got != "- [ ] b\n-\t[ ] a\n    cont\n" {
			t.Errorf("%v\n%q", err, got)
		}
	})
	t.Run("CRLF sin salto final", func(t *testing.T) {
		got, err := swap(t, "- [ ] a\r\n- [ ] b", 1, 2)
		if err != nil || got != "- [ ] b\r\n- [ ] a" {
			t.Errorf("las terminaciones se quedan en su posición: %v\n%q", err, got)
		}
		got, err = swap(t, "- [ ] a\n- [ ] b\r\n", 1, 2)
		if err != nil || got != "- [ ] b\n- [ ] a\r\n" {
			t.Errorf("mezcla original se conserva por posición: %v\n%q", err, got)
		}
	})
	t.Run("tabuladores y espacios", func(t *testing.T) {
		got, err := swap(t, "- [ ] p\n    - [ ] a\n\t- [ ] b\n", 2, 3) // 4 espacios y un tab: misma sangría
		if err != nil || got != "- [ ] p\n\t- [ ] b\n    - [ ] a\n" {
			t.Errorf("4 espacios y tab son la misma sangría: %v\n%q", err, got)
		}
		doc := "- [ ] p\n  - [ ] a\n\t- [ ] b\n" // 2 espacios y un tab (4): distintas
		if got, err := swap(t, doc, 2, 3); !errors.Is(err, ErrNotSiblings) || got != doc {
			t.Errorf("2 espacios y un tab no son hermanas: %v\n%q", err, got)
		}
	})
}

// TestSwapNumberedMixedTabsKeepsHierarchy (ORD-014, segunda opinión): en un bloque con líneas de espacios y de tab, al correr el contenido
// (9. → 10.) las de tab también se corren (después del tab) para no igualar niveles: la subtarea no pierde a su hija.
func TestSwapNumberedMixedTabsKeepsHierarchy(t *testing.T) {
	doc := "9. [ ] a\n   - [ ] s1\n\t - [ ] hija\n10. [ ] b\n"
	s, p := kanbanNote(t, doc)
	if _, _, err := s.SwapTasks(p, 1, 4, time.Time{}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	want := "9. [ ] b\n10. [ ] a\n    - [ ] s1\n\t  - [ ] hija\n"
	if string(b) != want {
		t.Errorf("\n got %q\nwant %q", b, want)
	}
	// la hija sigue colgando de s1 (goldmark): existe un ítem cuyo texto empieza en s1 y contiene a la hija
	var s1 string
	for _, li := range itemsHTML(t, string(b)) {
		if strings.Contains(li, "s1") && !strings.Contains(li, "> a\n") && !strings.Contains(li, "> b") {
			s1 = li
		}
	}
	if !strings.Contains(s1, "hija") {
		t.Errorf("s1 debe contener a su hija; ítems: %q", itemsHTML(t, string(b)))
	}
}
