package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// otherNotesDir crea otra carpeta de notas con contenido distinto y algunas
// carpetas y archivos que el selector no debe ofrecer o debe distinguir.
func otherNotesDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("alpha.md", "# Alpha\n\nNota de otra carpeta. #alpha\n\n- [ ] Tarea de alpha\n")
	write("sub/beta.md", "# Beta\n\n- [x] Tarea hecha de beta #beta\n")
	write(".oculta/secreta.md", "# Secreta\n")
	write("no-es-carpeta.txt", "texto")
	return dir
}

func savedNotesDir(t *testing.T, m *AppModel) string {
	t.Helper()
	data, err := os.ReadFile(m.c.cfg.Path())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestApplyNotesDir (H1-7b, C13): cambiar la carpeta de notas recarga el árbol, las
// tareas y las categorías sin reiniciar, y guarda la ruta.
func TestApplyNotesDir(t *testing.T) {
	m := newTestModel(t, 120, 35)
	original := m.c.store.BaseDir
	other := otherNotesDir(t)
	// estado de la carpeta anterior que no debe sobrevivir al cambio
	m.notes.setFilter("personal")
	m.notes.selected[filepath.Join(original, "compras.md")] = true
	m.notes.expanded[filepath.Join(original, "proyectos")] = true
	m.notes.list.set(1, len(m.notes.entries))
	m.tasks.list.set(3, len(m.c.tasks))
	m.tags.list.set(1, len(m.c.tags))

	m.applyNotesDir(other)

	if m.c.store.BaseDir != other || m.c.cfg.NotesDir != other {
		t.Fatalf("BaseDir=%q cfg=%q, se esperaba %q", m.c.store.BaseDir, m.c.cfg.NotesDir, other)
	}
	if len(m.c.notes) != 2 {
		t.Errorf("notas = %d, se esperaban las 2 de la carpeta nueva", len(m.c.notes))
	}
	out := plain(m)
	for _, want := range []string{"alpha.md", "sub", "Tarea de alpha", "#alpha"} {
		if !strings.Contains(out, want) {
			t.Errorf("tras el cambio falta %q en pantalla:\n%s", want, out)
		}
	}
	for _, gone := range []string{"compras.md", "bitacora.md", "Leche"} {
		if strings.Contains(out, gone) {
			t.Errorf("sigue apareciendo %q de la carpeta anterior", gone)
		}
	}
	if m.notes.tagFilter != "" || len(m.notes.selected) != 0 || len(m.notes.expanded) != 0 {
		t.Errorf("el panel de notas conserva estado de la carpeta anterior: filtro=%q selección=%v expandidas=%v", m.notes.tagFilter, m.notes.selected, m.notes.expanded)
	}
	if m.notes.list.cursor != 0 || m.tasks.list.cursor != 0 || m.tags.list.cursor != 0 {
		t.Errorf("los cursores no se reiniciaron: notas=%d tareas=%d categorías=%d", m.notes.list.cursor, m.tasks.list.cursor, m.tags.list.cursor)
	}
	if !strings.Contains(savedNotesDir(t, m), other) && runtime.GOOS != "windows" {
		t.Errorf("la ruta no se guardó en la configuración:\n%s", savedNotesDir(t, m))
	}
	if !strings.Contains(m.c.status, "(2") {
		t.Errorf("el estado no resume el cambio: %q", m.c.status)
	}

	// y se puede volver a la anterior
	m.applyNotesDir(original)
	if len(m.c.notes) < 5 || m.c.store.BaseDir != original {
		t.Errorf("no volvió a la carpeta original: %d notas en %q", len(m.c.notes), m.c.store.BaseDir)
	}
}

// TestApplyNotesDirRejectsInvalid: una ruta que no existe o no es una carpeta no cambia nada.
func TestApplyNotesDirRejectsInvalid(t *testing.T) {
	m := newTestModel(t, 120, 35)
	original, n := m.c.store.BaseDir, len(m.c.notes)
	other := otherNotesDir(t)
	for _, bad := range []string{filepath.Join(other, "no-existe"), filepath.Join(other, "no-es-carpeta.txt")} {
		m.applyNotesDir(bad)
		if m.c.store.BaseDir != original || m.c.cfg.NotesDir != original || len(m.c.notes) != n {
			t.Fatalf("%s: la carpeta cambió", bad)
		}
		if !strings.Contains(m.c.status, "Carpeta no válida") {
			t.Errorf("%s: no avisó: %q", bad, m.c.status)
		}
	}
}

func openPicker(t *testing.T, m *AppModel) (*settingsPopup, *folderPickerPopup) {
	t.Helper()
	press(m, ",")
	sp := m.c.top().(*settingsPopup)
	sp.list.set(int(setNotesDir), sp.n)
	press(m, "enter")
	fp, ok := m.c.top().(*folderPickerPopup)
	if !ok {
		t.Fatalf("la fila de carpeta de notas no abrió el selector: %T", m.c.top())
	}
	return sp, fp
}

func (p *folderPickerPopup) indexOf(name string) int {
	for i := 0; i < p.n; i++ {
		if p.name(i) == name {
			return i
		}
	}
	return -1
}

// TestFolderPickerKeyboard: desde Ajustes se recorre el disco con el teclado y `s`
// elige la carpeta; las ocultas no se ofrecen, ".." sube y deja el cursor donde estaba.
func TestFolderPickerKeyboard(t *testing.T) {
	m := newTestModel(t, 120, 35)
	other := otherNotesDir(t)
	m.applyNotesDir(other)
	sp, fp := openPicker(t, m)

	if fp.dir != other {
		t.Fatalf("el selector debe partir de la carpeta actual: %q", fp.dir)
	}
	if fp.indexOf(".oculta") >= 0 || fp.indexOf("no-es-carpeta.txt") >= 0 {
		t.Errorf("el selector ofrece carpetas ocultas o archivos: %v", fp.subdirs)
	}
	if fp.indexOf("..") != 0 || fp.indexOf("sub") != 1 {
		t.Fatalf("filas inesperadas: %v", fp.subdirs)
	}
	if out := plain(m); !strings.Contains(out, "[s] Usar esta carpeta") || !strings.Contains(out, "sub") {
		t.Errorf("el selector no muestra su contenido:\n%s", out)
	}

	fp.list.set(fp.indexOf("sub"), fp.n)
	press(m, "enter")
	if fp.dir != filepath.Join(other, "sub") || !fp.hasParent() {
		t.Fatalf("Enter debía entrar en sub: %q", fp.dir)
	}
	press(m, "left") // sube y deja el cursor sobre la carpeta de la que se vino
	if fp.dir != other || fp.name(fp.list.cursor) != "sub" {
		t.Fatalf("← debía subir con el cursor en sub: dir=%q fila=%q", fp.dir, fp.name(fp.list.cursor))
	}

	press(m, "enter") // entra otra vez
	press(m, "s")
	if _, ok := m.c.top().(*settingsPopup); !ok || m.c.top() != popup(sp) {
		t.Fatalf("tras elegir, el selector debe cerrarse y quedar Ajustes: %T", m.c.top())
	}
	want := filepath.Join(other, "sub")
	if m.c.cfg.NotesDir != want || len(m.c.notes) != 1 {
		t.Errorf("no se eligió sub: cfg=%q notas=%d", m.c.cfg.NotesDir, len(m.c.notes))
	}
	press(m, "esc")
	if out := plain(m); !strings.Contains(out, "beta.md") || strings.Contains(out, "alpha.md") {
		t.Errorf("el árbol no muestra la carpeta elegida:\n%s", out)
	}
}

// TestFolderPickerEscCancels: Esc cierra el selector sin cambiar nada.
func TestFolderPickerEscCancels(t *testing.T) {
	m := newTestModel(t, 120, 35)
	original := m.c.store.BaseDir
	_, fp := openPicker(t, m)
	fp.list.set(fp.indexOf("proyectos"), fp.n)
	press(m, "enter", "esc")
	if m.c.store.BaseDir != original || m.c.cfg.NotesDir != original {
		t.Error("cancelar no debe cambiar la carpeta")
	}
	if _, ok := m.c.top().(*settingsPopup); !ok {
		t.Errorf("tras cancelar debe quedar Ajustes: %T", m.c.top())
	}
}

// TestFolderPickerMouse: con el mouse, el primer clic marca una carpeta, el segundo
// entra, y el botón elige la carpeta actual.
func TestFolderPickerMouse(t *testing.T) {
	m := newTestModel(t, 120, 35)
	other := otherNotesDir(t)
	m.applyNotesDir(other)
	_, fp := openPicker(t, m)
	r := popupRect(m.layout, fp, fp.render(m.layout))

	row := r.Y + fp.top + fp.indexOf("sub")
	click(m, r.X+6, row)
	if fp.dir != other || fp.list.cursor != fp.indexOf("sub") {
		t.Fatalf("el 1.er clic solo marca: dir=%q cursor=%d", fp.dir, fp.list.cursor)
	}
	click(m, r.X+6, row)
	if fp.dir != filepath.Join(other, "sub") {
		t.Fatalf("el 2.º clic entra: dir=%q", fp.dir)
	}
	r = popupRect(m.layout, fp, fp.render(m.layout))
	click(m, r.X+4, r.Y+fp.buttonRow)
	if m.c.cfg.NotesDir != filepath.Join(other, "sub") {
		t.Errorf("el botón no eligió la carpeta: %q", m.c.cfg.NotesDir)
	}
	if _, ok := m.c.top().(*settingsPopup); !ok {
		t.Errorf("el selector debía cerrarse: %T", m.c.top())
	}
}

// TestFolderPickerRoot: en la raíz del disco no hay fila "..".
func TestFolderPickerRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows la raíz es una unidad")
	}
	fp := newFolderPicker("/", func(string) tea.Cmd { return nil })
	if fp.hasParent() || fp.indexOf("..") >= 0 {
		t.Error("la raíz no tiene carpeta superior")
	}
}

// TestSettingsShowsNotesDir: Ajustes muestra la carpeta actual abreviada con ~.
func TestSettingsShowsNotesDir(t *testing.T) {
	m := newTestModel(t, 120, 35)
	home, _ := os.UserHomeDir()
	t.Setenv("HOME", home)
	press(m, ",")
	if out := plain(m); !strings.Contains(out, "Carpeta de notas") {
		t.Errorf("Ajustes no tiene la fila de la carpeta de notas:\n%s", out)
	}
}

func TestShortPathAndLeftTruncate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	sep := string(filepath.Separator)
	if got := shortPath(filepath.Join(home, "Documents", "notes")); got != "~"+sep+"Documents"+sep+"notes" {
		t.Errorf("shortPath dentro de HOME = %q", got)
	}
	if got := shortPath(home); got != "~" {
		t.Errorf("shortPath(HOME) = %q", got)
	}
	outside := filepath.Join(t.TempDir(), "x")
	if got := shortPath(outside); got != outside {
		t.Errorf("shortPath fuera de HOME = %q", got)
	}
	if got := leftTruncate("/una/ruta/muy/larga/de/notas", 12); !strings.HasPrefix(got, "…") || len([]rune(got)) != 12 || !strings.HasSuffix(got, "de/notas") {
		t.Errorf("leftTruncate = %q", got)
	}
	if got := leftTruncate("/corta", 12); got != "/corta" {
		t.Errorf("leftTruncate no debe tocar lo que cabe: %q", got)
	}
}
