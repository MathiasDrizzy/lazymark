package app

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/charmbracelet/x/ansi"
)

// copyFixtures copia testdata/notes a un directorio temporal para que el test
// nunca toque las notas reales ni modifique los fixtures versionados. Las fechas
// de modificación se fijan explícitamente (cada archivo, alfabéticamente, un
// minuto después del anterior): el orden de las notas y de sus tareas no puede
// depender del sistema de archivos ni de la velocidad de la copia.
// Con LAZYMARK_TEST_CHAOS=<semilla> se barajan, para comprobar que ningún test
// depende de ese orden.
func copyFixtures(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("testdata", "notes")
	var files []string
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, target)
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copiando fixtures: %v", err)
	}
	order := make([]int, len(files))
	for i := range order {
		order[i] = i
	}
	if seed, err := strconv.Atoi(os.Getenv("LAZYMARK_TEST_CHAOS")); err == nil {
		rand.New(rand.NewSource(int64(seed))).Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i, f := range files {
		mt := base.Add(time.Duration(order[i]) * time.Minute)
		if err := os.Chtimes(f, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// newTestModel crea el modelo con fixtures, con HOME aislado (Save nunca
// escribe la config real) y con el tamaño de terminal dado.
func newTestModel(t *testing.T, w, h int) *AppModel {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData")) // Windows
	cfg := config.DefaultConfig(copyFixtures(t))
	cfg.Language = "es"
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// R17: los tests nunca leen el portapapeles real; el que lo necesite pone un lector simulado.
	m.c.clip.Reader = noRealClipboard{}
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

// noRealClipboard hace fallar cualquier lectura del portapapeles del sistema.
type noRealClipboard struct{}

func (noRealClipboard) ReadImage(string) error {
	return errors.New("portapapeles real deshabilitado en los tests")
}

// press envía teclas como si las escribiera el usuario ("enter", "space", "?"…).
func press(m *AppModel, keys ...string) {
	for _, k := range keys {
		m.Update(keyMsg(k))
	}
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: r, Text: k}
}

func screen(m *AppModel) []string { return strings.Split(m.View().Content, "\n") }

func plain(m *AppModel) string { return ansi.Strip(m.View().Content) }

// TestViewFitsTerminal (X4, X5): View() nunca desborda la terminal; cada línea
// cabe en el ancho y la altura es exactamente la de la terminal.
func TestViewFitsTerminal(t *testing.T) {
	sizes := []struct{ w, h int }{{120, 35}, {100, 30}, {80, 24}, {60, 20}}
	states := []struct {
		name string
		keys []string
	}{
		{"base", nil},
		{"cheatsheet", []string{"?"}},
		{"settings", []string{","}},
		{"papelera", []string{"x"}},
		{"renombrar", []string{"r"}},
		{"mover", []string{"m"}},
		{"borrar", []string{"d"}},
		{"preview-enfocado", []string{"4"}},
		{"tareas", []string{"2"}},
		{"tags", []string{"3"}},
		{"zoom", []string{"4", "w"}},
		{"kanban", []string{"W"}},
		{"kanban-cheatsheet", []string{"W", "?"}},
	}
	for _, sz := range sizes {
		for _, st := range states {
			t.Run(fmt.Sprintf("%s/%dx%d", st.name, sz.w, sz.h), func(t *testing.T) {
				m := newTestModel(t, sz.w, sz.h)
				press(m, st.keys...)
				lines := screen(m)
				if len(lines) != sz.h {
					t.Errorf("altura = %d líneas, se esperaban %d", len(lines), sz.h)
				}
				for i, line := range lines {
					if w := ansi.StringWidth(line); w != sz.w {
						t.Errorf("línea %d: ancho %d != %d: %q", i, w, sz.w, ansi.Strip(line))
					}
				}
			})
		}
	}
}

// TestLongNamesKeepColumns (X5): un nombre largo con emoji y NerdFont se corta
// con "…" y la fecha queda en la misma columna en todas las filas.
func TestLongNamesKeepColumns(t *testing.T) {
	m := newTestModel(t, 120, 35)
	r := m.layout.Notes
	lines := screen(m)[r.Y+1 : r.Y+r.H-1]
	dateCol := -1
	sawLong := false
	for _, l := range lines {
		s := ansi.Strip(l)
		if strings.Contains(s, "larguísimo") || strings.Contains(s, "alerta") {
			sawLong = true
			if !strings.Contains(s, "…") {
				t.Errorf("el nombre largo no se cortó con …: %q", s)
			}
		}
		if i := strings.LastIndex(s, " "); strings.Count(s, "Jan")+strings.Count(s, "Feb")+strings.Count(s, "Oct") > 0 && i > 0 {
			col := ansi.StringWidth(s[:i])
			if dateCol == -1 {
				dateCol = col
			} else if col != dateCol {
				t.Errorf("la fecha cambió de columna: %d vs %d en %q", col, dateCol, s)
			}
		}
	}
	if !sawLong {
		t.Fatalf("no apareció la nota de nombre largo en el árbol:\n%s", strings.Join(lines, "\n"))
	}
}

// TestPopupKeepsRightFrame (H0-3): con el cheatsheet abierto, el borde derecho
// de los paneles de fondo sigue intacto.
func TestPopupKeepsRightFrame(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{120, 35}, {100, 30}, {80, 24}} {
		for _, kanban := range []bool{false, true} {
			t.Run(fmt.Sprintf("kanban=%v/%dx%d", kanban, sz.w, sz.h), func(t *testing.T) {
				m := newTestModel(t, sz.w, sz.h)
				if kanban {
					press(m, "W")
				}
				base := screen(m)
				press(m, "?")
				with := screen(m)
				for y := 0; y < sz.h-1; y++ {
					if a, b := lastCell(base[y]), lastCell(with[y]); a != b {
						t.Errorf("fila %d: borde derecho %q con popup, %q sin popup", y, b, a)
					}
				}
			})
		}
	}
}

func lastCell(line string) string {
	w := ansi.StringWidth(line)
	return ansi.Strip(ansi.Cut(line, w-1, w))
}

// TestTooSmallWarning (X6): por debajo de 60x20 se avisa en vez de dibujar.
func TestTooSmallWarning(t *testing.T) {
	for _, sz := range []struct{ w, h int }{{59, 30}, {100, 19}, {40, 10}} {
		m := newTestModel(t, sz.w, sz.h)
		out := plain(m)
		if !strings.Contains(out, "Terminal muy pequeña") || !strings.Contains(out, "Mínimo 60x20") {
			t.Errorf("%dx%d sin aviso:\n%s", sz.w, sz.h, out)
		}
		if lines := screen(m); len(lines) != sz.h {
			t.Errorf("%dx%d: %d líneas", sz.w, sz.h, len(lines))
		}
	}
	if out := plain(newTestModel(t, 60, 20)); strings.Contains(out, "Terminal muy pequeña") {
		t.Error("60x20 es el mínimo y no debería avisar")
	}
}
