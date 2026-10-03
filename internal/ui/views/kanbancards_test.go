package views

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
)

func dateBoard() KanbanBoard {
	notes := []storage.Note{{
		ID: "n.md", Title: "Plan", Path: "/notes/plan.md",
		Tasks: []storage.Task{
			{NoteTitle: "Plan", Line: 3, Text: "Vencida 📅 2020-01-02 🛫 2019-12-01", Dates: storage.ParseDates("📅 2020-01-02 🛫 2019-12-01")},
			{NoteTitle: "Plan", Line: 4, Text: "Futura 📅 2099-01-01", Dates: storage.ParseDates("📅 2099-01-01")},
			{NoteTitle: "Plan", Line: 5, Text: "Sin fechas"},
			{NoteTitle: "Plan", Line: 6, Text: "Texto muy largo que no cabe en una sola línea y necesita partirse en dos líneas para que se pueda leer bien en la tarjeta del tablero", Done: false},
			{NoteTitle: "Plan", Line: 7, Text: "Hecha 📅 2020-01-02 ✅ 2020-01-01", Done: true, Dates: storage.ParseDates("📅 2020-01-02 ✅ 2020-01-01")},
		},
	}}
	return CollectKanban(notes, storage.DefaultColumns, []string{"To do", "In progress", "Done"})
}

var sgrRe = regexp.MustCompile("\x1b\\[([0-9;]*)m")

// segment es un tramo de texto con los parámetros SGR que lo preceden.
type segment struct{ params, text string }

// sgrSegments parte s en tramos de texto, cada uno con el último SGR que lo precede (suficiente para ver colores de primer plano).
func sgrSegments(s string) []segment {
	var out []segment
	params, pos := "", 0
	for _, m := range sgrRe.FindAllStringSubmatchIndex(s, -1) {
		if m[0] > pos {
			out = append(out, segment{params, s[pos:m[0]]})
		}
		params, pos = s[m[2]:m[3]], m[1]
	}
	if pos < len(s) {
		out = append(out, segment{params, s[pos:]})
	}
	return out
}

// hasFg indica si el tramo está en el color de primer plano c (truecolor: 38;2;r;g;b).
func (sg segment) hasFg(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return strings.Contains(sg.params, fmt.Sprintf("38;2;%d;%d;%d", r>>8, g>>8, b>>8))
}

// TestRenderCards (C.6): cada tarea es un rectángulo de borde redondeado con su texto (hasta 2 líneas, con …), su nota de
// origen y sus fechas; el vencimiento de una tarea vencida va en el color de error del tema y el de una hecha o futura no.
func TestRenderCards(t *testing.T) {
	ht := mouse.NewHitTester()
	out := RenderKanban(dateBoard(), 0, []int{0, 0, 0}, 120, 35, ht, 1, KanbanDrag{}, KanbanOptions{Cards: true, Today: "2026-10-02"})
	plain := ansi.Strip(out)
	lines := strings.Split(plain, "\n")
	if len(lines) != 35 {
		t.Fatalf("%d filas, se esperaban 35", len(lines))
	}
	for i, l := range lines {
		if w := textwidth.Width(l); w != 120 {
			t.Errorf("la fila %d mide %d columnas", i, w)
		}
	}
	for _, want := range []string{"╭──", "╰──", "│ ▸ ☐ Vencida", "· Plan", "🛫 2019-12-01 📅 2020-01-02", "📅 2099-01-01", "☐ Sin fechas", "📅 2020-01-02 ✅ 2020-01-01"} {
		if !strings.Contains(plain, want) {
			t.Errorf("falta %q:\n%s", want, plain)
		}
	}
	if !strings.Contains(plain, "en una sola línea") || !strings.Contains(plain, "…") || strings.Contains(plain, "tarjeta del tablero") {
		t.Errorf("el texto largo debe partirse en 2 líneas y cortarse con …:\n%s", plain)
	}
	// el color de error: solo en la fecha de la tarea vencida
	redDates := 0
	for _, seg := range sgrSegments(out) {
		if seg.hasFg(theme.ColorRed) {
			if strings.Contains(seg.text, "2020-01-02") {
				redDates++
			} else if strings.TrimSpace(seg.text) != "" {
				t.Errorf("algo más va en rojo: %q", seg.text)
			}
		}
	}
	if redDates != 1 {
		t.Errorf("la fecha vencida debe ir en rojo una vez (la de la hecha no): %d", redDates)
	}
	// la zona de clic cubre todo el alto de la tarjeta (borde incluido), no solo la fila del texto
	z0, ok := ht.Check(10, 2) // la fila del texto de la primera tarjeta
	if !ok || z0.Type != mouse.ZoneKanbanCard {
		t.Fatalf("la fila del texto debe ser una zona de tarjeta: %v %+v", ok, z0)
	}
	for y := 2; y <= 6; y++ { // borde de arriba, texto, nota, fechas y borde de abajo
		if z, ok := ht.Check(10, y); !ok || z.Payload != z0.Payload {
			t.Errorf("la fila %d debe ser de la misma tarjeta: %v %+v", y, ok, z)
		}
	}
	if z, ok := ht.Check(10, 7); ok && z.Payload == z0.Payload {
		t.Error("la fila de abajo de la tarjeta ya es de otra")
	}
	if !strings.HasSuffix(z0.Payload, "|/notes/plan.md") || !strings.HasPrefix(z0.Payload, "0|3|") {
		t.Errorf("payload %q", z0.Payload)
	}
}

// TestCardsFallBackToCompact (C.6): la opción "compactas" y las columnas o el tablero demasiado chicos usan la vista de una fila.
func TestCardsFallBackToCompact(t *testing.T) {
	b := dateBoard()
	for name, c := range map[string]struct {
		w, h int
		opts KanbanOptions
	}{
		"opción compactas": {120, 35, KanbanOptions{Cards: false}},
		"columna angosta":  {66, 35, KanbanOptions{Cards: true}},
		"tablero bajo":     {120, 8, KanbanOptions{Cards: true}},
	} {
		plain := ansi.Strip(RenderKanban(b, 0, []int{0, 0, 0}, c.w, c.h, nil, 0, KanbanDrag{}, c.opts))
		if strings.Contains(plain, "╭──") && strings.Count(plain, "╭") > 3 {
			t.Errorf("%s: debía ser la vista compacta:\n%s", name, plain)
		}
		if !strings.Contains(plain, "☐ Vencida") {
			t.Errorf("%s: falta la tarea:\n%s", name, plain)
		}
	}
}

// TestCardsScrollKeepSelectionVisible: con más tarjetas de las que caben, la seleccionada siempre se ve.
func TestCardsScrollKeepSelectionVisible(t *testing.T) {
	var tasks []storage.Task
	for i := 1; i <= 20; i++ {
		tasks = append(tasks, storage.Task{NoteTitle: "N", Line: i, Text: "tarea número " + strings.Repeat("x", i%3)})
	}
	tasks[17].Text = "LA_ELEGIDA"
	b := CollectKanban([]storage.Note{{Title: "N", Path: "/n.md", Tasks: tasks}}, storage.DefaultColumns, []string{"a", "b", "c"})
	plain := ansi.Strip(RenderKanban(b, 0, []int{17, 0, 0}, 120, 20, nil, 0, KanbanDrag{}, KanbanOptions{Cards: true}))
	if !strings.Contains(plain, "▸ ☐ LA_ELEGIDA") {
		t.Errorf("la tarjeta seleccionada debe verse:\n%s", plain)
	}
	if !strings.Contains(plain, "18 of 20") {
		t.Errorf("el contador de la columna:\n%s", plain)
	}
}

// TestWrapLines: parte por palabras hasta 2 líneas, corta con … lo que sobra y los textos sin espacios (japonés) por caracteres.
func TestWrapLines(t *testing.T) {
	for _, c := range []struct {
		in   string
		w, n int
		want []string
	}{
		{"hola mundo", 20, 2, []string{"hola mundo"}},
		{"uno dos tres cuatro", 9, 2, []string{"uno dos", "tres cua…"}},
		{"", 10, 2, []string{""}},
		{"palabraenormesinespacios", 8, 2, []string{"palabrae", "normesi…"}},
	} {
		got := wrapLines(c.in, c.w, c.n)
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("wrapLines(%q,%d,%d) = %q, se esperaba %q", c.in, c.w, c.n, got, c.want)
		}
	}
	got := wrapLines("資料を書いてチームに送る大事な作業の説明文です", 12, 2)
	if len(got) != 2 || textwidth.Width(got[0]) > 12 || textwidth.Width(got[1]) > 12 || !strings.HasSuffix(got[1], "…") {
		t.Errorf("japonés: %q", got)
	}
}

// TestCardHeightMatchesRender: el alto con el que se calculan el scroll y las zonas de clic es el de la tarjeta dibujada, para
// cualquier ancho y largo de texto (un desajuste dejaba filas de la tarjeta sin zona y corría los clics de las siguientes).
func TestCardHeightMatchesRender(t *testing.T) {
	for w := 24; w <= 60; w++ {
		for n := 1; n <= 70; n++ {
			c := KanbanCard{CleanText: strings.TrimSpace(strings.Repeat("palabra ", n/8+1)[:n]), NoteTitle: "N"}
			if got, want := len(renderCard(c, w, false, false, false, true, false, "2026-01-01")), cardHeight(c, w); got != want {
				t.Fatalf("ancho %d, texto de %d caracteres (%q): se dibujan %d filas y cardHeight dice %d", w, n, c.CleanText, got, want)
			}
		}
	}
}
