package app

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/charmbracelet/x/ansi"
)

// TestLayoutInEveryLanguage (C.5): en cada idioma (también ja y zh, de caracteres de doble ancho), a 120x35 y a 80x24,
// ninguna pantalla se desborda: tiene exactamente las filas de la terminal, cada fila mide exactamente su ancho, y los
// bordes de los paneles caen en las columnas del layout (un carácter ancho cortado o de más los correría).
func TestLayoutInEveryLanguage(t *testing.T) {
	prev := i18n.CurrentLanguage()
	defer i18n.SetLanguage(string(prev))
	states := []struct {
		name string
		keys []string
	}{{"notas", nil}, {"kanban", []string{"W"}}, {"ajustes", []string{","}}, {"atajos", []string{"?"}}, {"papelera", []string{"x"}}, {"mover", []string{"m"}}}
	for _, lang := range i18n.Languages {
		for _, sz := range []struct{ w, h int }{{120, 35}, {80, 24}} {
			for _, st := range states {
				m := newTestModel(t, sz.w, sz.h)
				i18n.SetLanguage(string(lang)) // después de crear el modelo: New aplica el idioma de la config
				m.relayout()
				press(m, st.keys...)
				if got := i18n.CurrentLanguage(); got != lang {
					t.Fatalf("el idioma activo es %s, no %s", got, lang)
				}
				out := screen(m)
				if len(out) != sz.h {
					t.Errorf("%s %dx%d %s: %d filas, se esperaban %d", lang, sz.w, sz.h, st.name, len(out), sz.h)
				}
				for y, l := range out {
					if w := ansi.StringWidth(l); w != sz.w {
						t.Errorf("%s %dx%d %s: la fila %d mide %d columnas, se esperaban %d: %q", lang, sz.w, sz.h, st.name, y, w, sz.w, ansi.Strip(l))
						break
					}
				}
				if st.name == "notas" {
					top := gridOf(m.View().Content, sz.w, sz.h)
					for _, x := range []int{0, m.layout.Preview.X} { // esquinas superiores izquierdas de los dos bloques
						if c := top.CellAt(x, 0); c == nil || c.Content != "╭" {
							t.Errorf("%s %dx%d: falta la esquina ╭ del panel en x=%d", lang, sz.w, sz.h, x)
						}
					}
					if c := top.CellAt(sz.w-1, 0); c == nil || c.Content != "╮" {
						t.Errorf("%s %dx%d: falta la esquina ╮ del último panel", lang, sz.w, sz.h)
					}
					for y := 1; y < sz.h-2; y++ { // el borde derecho de la pantalla, fila a fila
						if c := top.CellAt(sz.w-1, y); c == nil || !strings.ContainsAny(c.Content, "│╮╯") {
							t.Errorf("%s %dx%d: el borde derecho se corrió en la fila %d: %v", lang, sz.w, sz.h, y, c)
							break
						}
					}
				}
			}
		}
	}
}

// TestStatusPluralAndKanbanButton (C.7): "1 nota cargada" en singular, y el botón "Kanban (W)" de la barra y el de volver
// ("Notas (W)") se traducen (カンバン en japonés, 看板 en chino).
func TestStatusPluralAndKanbanButton(t *testing.T) {
	defer i18n.SetLanguage("es")
	for _, tc := range []struct {
		lang, one, many, button string
	}{
		{"es", "1 nota cargada", "3 notas cargadas", "Kanban (W)"},
		{"en", "1 note loaded", "3 notes loaded", "Kanban (W)"},
		{"ja", "1 件のノートを読み込みました", "", "カンバン (W)"},
		{"zh", "已加载 1 篇笔记", "", "看板 (W)"},
	} {
		i18n.SetLanguage(tc.lang)
		m := newTestModel(t, 120, 35)
		i18n.SetLanguage(tc.lang)
		m.relayout()
		if !strings.Contains(plain(m), tc.button) {
			t.Errorf("%s: falta el botón %q en la barra:\n%s", tc.lang, tc.button, plain(m))
		}
	}
	for _, tc := range []struct {
		lang string
		n    int
		want string
	}{{"es", 1, "1 nota cargada"}, {"es", 3, "3 notas cargadas"}, {"en", 1, "1 note loaded"}, {"en", 2, "2 notes loaded"}} {
		i18n.SetLanguage(tc.lang)
		if got := loadedStatus(tc.n); got != tc.want {
			t.Errorf("%s %d: %q, se esperaba %q", tc.lang, tc.n, got, tc.want)
		}
	}
}
