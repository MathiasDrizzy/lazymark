package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// typeText escribe texto en el popup de nombre, borrando antes el sugerido.
func typeText(m *AppModel, s string) {
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func cursorPath(m *AppModel) string {
	if e := m.notes.current(); e != nil {
		return e.Path
	}
	return ""
}

func click(m *AppModel, x, y int) {
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
}

// TestCreateRenameMove (H1-2): crear deja el cursor sobre lo creado y no abre
// el editor; renombrar y mover conservan el cursor sobre el elemento.
func TestCreateRenameMove(t *testing.T) {
	m := newTestModel(t, 120, 35)
	base := m.c.store.BaseDir

	press(m, "G", "c")
	if _, ok := m.c.top().(*inputPopup); !ok {
		t.Fatal("c no abrió el popup de nombre")
	}
	if !strings.Contains(plain(m), "Nombre") {
		t.Error("el popup de nombre no dice 'Nombre'")
	}
	typeText(m, "Nota H1")
	press(m, "enter")
	created := filepath.Join(base, "nota-h1.md")
	if m.c.top() != nil {
		t.Fatal("el popup no se cerró al crear")
	}
	if got := cursorPath(m); got != created {
		t.Fatalf("cursor en %q, se esperaba la nota creada %q", got, created)
	}

	press(m, "F")
	typeText(m, "Archivo")
	press(m, "enter")
	folder := filepath.Join(base, "archivo")
	if got := cursorPath(m); got != folder {
		t.Fatalf("cursor en %q, se esperaba la carpeta creada", got)
	}

	m.notes.selectPath(created)
	press(m, "r")
	typeText(m, "renombrada")
	press(m, "enter")
	renamed := filepath.Join(base, "renombrada.md")
	if got := cursorPath(m); got != renamed {
		t.Fatalf("tras renombrar el cursor está en %q", got)
	}

	press(m, "m")
	mp, ok := m.c.top().(*movePopup)
	if !ok {
		t.Fatal("m no abrió el popup de mover")
	}
	for i, f := range mp.folders {
		if f == folder {
			mp.list.set(i, mp.n)
		}
	}
	press(m, "enter")
	moved := filepath.Join(folder, "renombrada.md")
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("la nota no se movió: %v", err)
	}
	if got := cursorPath(m); got != moved {
		t.Errorf("tras mover el cursor está en %q", got)
	}
}

// TestDeleteSelectionAndFolder (H1-3): la selección múltiple borra varias
// notas tras confirmar; una carpeta con contenido siempre pide confirmación.
func TestDeleteSelectionAndFolder(t *testing.T) {
	m := newTestModel(t, 120, 35)
	m.c.cfg.ConfirmDelete = false
	base := m.c.store.BaseDir

	m.notes.selectPath(filepath.Join(base, "compras.md"))
	press(m, "v")
	m.notes.selectPath(filepath.Join(base, "bienvenida.md"))
	press(m, "v", "d")
	for _, n := range []string{"compras.md", "bienvenida.md"} {
		if _, err := os.Stat(filepath.Join(base, n)); !os.IsNotExist(err) {
			t.Errorf("%s no se borró con la selección múltiple", n)
		}
	}
	if m.c.trashCount != 2 {
		t.Errorf("trashCount = %d", m.c.trashCount)
	}

	m.notes.selectPath(filepath.Join(base, "proyectos"))
	press(m, "d")
	if _, ok := m.c.top().(*confirmPopup); !ok {
		t.Fatal("borrar una carpeta con contenido no pidió confirmación aunque ConfirmDelete=false")
	}
	press(m, "n")
	if _, err := os.Stat(filepath.Join(base, "proyectos")); err != nil {
		t.Fatal("cancelar borró la carpeta")
	}
	press(m, "d", "y")
	if _, err := os.Stat(filepath.Join(base, "proyectos")); !os.IsNotExist(err) {
		t.Error("confirmar no borró la carpeta")
	}
}

// TestSettingsClicks (H1-7): el primer clic mueve '>' y el segundo cambia el valor.
func TestSettingsClicks(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, ",")
	p := m.c.top().(*settingsPopup)
	r := popupRect(m.layout, p, p.render(m.layout))
	row := r.Y + p.top + int(setConfirmDelete)
	before := m.c.cfg.ConfirmDelete

	click(m, r.X+5, row)
	if p.list.cursor != int(setConfirmDelete) || m.c.cfg.ConfirmDelete != before {
		t.Fatalf("1.er clic: cursor=%d confirm=%v", p.list.cursor, m.c.cfg.ConfirmDelete)
	}
	click(m, r.X+5, row)
	if m.c.cfg.ConfirmDelete == before {
		t.Fatal("el 2.º clic no cambió el valor")
	}
	press(m, "left")
	if m.c.cfg.ConfirmDelete != before {
		t.Error("← no volvió al valor anterior")
	}
	click(m, 0, 0) // fuera del popup: se ignora
	if m.c.top() == nil || m.focus != panelNotes {
		t.Error("un clic fuera cerró el popup o cambió el foco")
	}
	press(m, "esc")
	if m.c.top() != nil {
		t.Error("Esc no cerró settings")
	}
}

// TestCheatsheetMatchesKeymap (X9, H1-8): el cheatsheet muestra exactamente los
// atajos del keymap para el contexto, con las teclas decididas en G.
func TestCheatsheetMatchesKeymap(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "?")
	p, ok := m.c.top().(*cheatsheetPopup)
	if !ok {
		t.Fatal("? no abrió el cheatsheet")
	}
	r := popupRect(m.layout, p, p.render(m.layout))
	if r.X+r.W != m.w-1 || r.Y+r.H != m.layout.Footer.Y-1 {
		t.Errorf("el cheatsheet no está abajo a la derecha: %+v", r)
	}
	out := ansi.Strip(p.render(m.layout))
	n := 0
	for _, sec := range p.sections() {
		for _, b := range sec {
			n++
			if !strings.Contains(out, b.Desc()) || !strings.Contains(out, b.KeyLabel()) {
				t.Errorf("falta %q (%s) en el cheatsheet", b.Desc(), b.KeyLabel())
			}
		}
	}
	if n != len(m.c.keys.In(ctxNotes))+len(m.c.keys.In(ctxNav))+len(m.c.keys.In(ctxGlobal)) {
		t.Errorf("el cheatsheet tiene %d atajos, distinto del keymap", n)
	}
	for key, want := range map[string]Action{"?": actCheatsheet, ",": actSettings, "x": actTrash, "q": actQuit} {
		if got := m.c.keys.Lookup(key, ctxGlobal); got != want {
			t.Errorf("tecla %q -> %v, se esperaba %v", key, got, want)
		}
	}
	if m.c.keys.Lookup("h", ctxNav) != actLeft || m.c.keys.Lookup("l", ctxNav) != actRight {
		t.Error("h/l no son izquierda/derecha (compatibilidad Vim)")
	}
	if m.c.keys.Lookup("p", ctxNotes, ctxNav, ctxGlobal) != actNone {
		t.Error("p no debería tener acción (H3-2)")
	}
	press(m, "?")
	if m.c.top() != nil {
		t.Error("? no cerró el cheatsheet")
	}
}

// TestTagFilter (H1-9): Enter o clic sobre un tag filtra el árbol.
func TestTagFilter(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "3")
	for i, tg := range m.c.tags {
		if tg.Name == "personal" {
			m.tags.list.set(i, len(m.c.tags))
		}
	}
	press(m, "enter")
	if m.notes.tagFilter != "personal" || len(m.notes.entries) != 1 || m.focus != panelNotes {
		t.Fatalf("Enter: filtro=%q entradas=%d foco=%v", m.notes.tagFilter, len(m.notes.entries), m.focus)
	}
	press(m, "esc")
	if m.notes.tagFilter != "" {
		t.Fatal("Esc no quitó el filtro")
	}
	r := m.layout.Tags
	click(m, r.X+3, r.Y+1)
	if m.notes.tagFilter != m.c.tags[0].Name {
		t.Errorf("clic en el primer tag: filtro=%q", m.notes.tagFilter)
	}
}

// TestDividerDrag (§4): arrastrar el divisor cambia la proporción de columnas.
func TestDividerDrag(t *testing.T) {
	m := newTestModel(t, 120, 35)
	before := m.layout.Preview.X
	d := m.layout.Divider
	click(m, d.X, 10)
	m.Update(tea.MouseMotionMsg{X: 60, Y: 10, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: 60, Y: 10, Button: tea.MouseLeft})
	if m.layout.Preview.X == before || m.layout.Preview.X < 55 {
		t.Errorf("el divisor no se movió: antes %d, después %d", before, m.layout.Preview.X)
	}
}

// TestFolderPreview: con el cursor sobre una carpeta, el preview lista sus notas.
func TestFolderPreview(t *testing.T) {
	m := newTestModel(t, 120, 35)
	m.notes.selectPath(filepath.Join(m.c.store.BaseDir, "proyectos"))
	out := plain(m)
	if !strings.Contains(out, "lazymark-roadmap.md") || !strings.Contains(out, "1 nota(s)") {
		t.Errorf("el preview de la carpeta no lista sus notas:\n%s", out)
	}
}

// TestMovePopupClickSelects (decisión G): en el popup de mover, un clic (o dos)
// solo selecciona la carpeta; Enter confirma el movimiento.
func TestMovePopupClickSelects(t *testing.T) {
	m := newTestModel(t, 120, 35)
	note := filepath.Join(m.c.store.BaseDir, "compras.md")
	m.notes.selectPath(note)
	press(m, "m")
	p := m.c.top().(*movePopup)
	r := popupRect(m.layout, p, p.render(m.layout))
	row := r.Y + p.top + 1
	click(m, r.X+4, row)
	click(m, r.X+4, row)
	if m.c.top() == nil {
		t.Fatal("los clics confirmaron el movimiento")
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal("la nota se movió sin Enter")
	}
	if p.list.cursor != 1 {
		t.Errorf("el clic no seleccionó la fila: cursor=%d", p.list.cursor)
	}
	press(m, "enter")
	if _, err := os.Stat(note); !os.IsNotExist(err) {
		t.Error("Enter no movió la nota")
	}
}
