package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// pngBytes devuelve el PNG de los fixtures.
func pngBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "notes", "assets", "foto.png"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeClip simula el portapapeles del sistema (R17: nunca el real).
type fakeClip struct {
	data  []byte
	err   error
	calls int
}

func (f *fakeClip) ReadImage(dest string) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	return os.WriteFile(dest, f.data, 0o644)
}

func ctrlV(m *AppModel) { m.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl}) }

func readNote(t *testing.T, m *AppModel, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(m.c.store.BaseDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func assetFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, "assets", "*"))
	return m
}

// TestCtrlVAppendsImage (H3-2, C3): Ctrl+V guarda la imagen del portapapeles en
// assets/ de la carpeta de la nota y deja `![](assets/…)` al final de la nota, sin
// tocar nada de lo anterior; el cursor sigue en la nota.
func TestCtrlVAppendsImage(t *testing.T) {
	m := newImageRig(t, 120, 35, true).AppModel
	clip := &fakeClip{data: pngBytes(t)}
	m.c.clip.Reader = clip
	base := m.c.store.BaseDir
	before := readNote(t, m, "compras.md")
	m.notes.selectPath(filepath.Join(base, "compras.md"))

	ctrlV(m)

	files := assetFiles(t, base)
	var mine string
	for _, f := range files {
		if strings.HasPrefix(filepath.Base(f), "compras-") {
			mine = filepath.Base(f)
		}
	}
	if mine == "" {
		t.Fatalf("no se guardó la imagen en assets/: %v", files)
	}
	after := readNote(t, m, "compras.md")
	if want := before + "\n![](assets/" + mine + ")\n"; after != want {
		t.Errorf("la nota debía quedar igual más la referencia al final:\n got %q\nwant %q", after, want)
	}
	if got := cursorPath(m); got != filepath.Join(base, "compras.md") {
		t.Errorf("el cursor se movió a %q", got)
	}
	if !strings.Contains(m.c.status, "assets/"+mine) {
		t.Errorf("el estado no dice dónde se guardó: %q", m.c.status)
	}
	if b, _ := os.ReadFile(filepath.Join(base, "assets", mine)); string(b) != string(clip.data) {
		t.Error("la imagen guardada no es la del portapapeles")
	}
}

// TestCtrlVSubfolderUsesItsOwnAssets: la imagen va a assets/ de la carpeta de la
// nota, que es donde la resuelve el preview.
func TestCtrlVSubfolderUsesItsOwnAssets(t *testing.T) {
	m := newTestModel(t, 120, 35)
	m.c.clip.Reader = &fakeClip{data: pngBytes(t)}
	base := m.c.store.BaseDir
	m.notes.selectPath(filepath.Join(base, "proyectos", "lazymark-roadmap.md"))
	ctrlV(m)
	if got := assetFiles(t, filepath.Join(base, "proyectos")); len(got) != 1 {
		t.Fatalf("assets de la subcarpeta = %v", got)
	}
	if !strings.Contains(readNote(t, m, filepath.Join("proyectos", "lazymark-roadmap.md")), "![](assets/lazymark-roadmap-") {
		t.Error("falta la referencia en la nota de la subcarpeta")
	}
}

// TestCtrlVBelowSelectedTask (decisión G): con el foco en Tareas, la referencia
// queda justo debajo de la tarea seleccionada.
func TestCtrlVBelowSelectedTask(t *testing.T) {
	m := newTestModel(t, 120, 35)
	m.c.clip.Reader = &fakeClip{data: pngBytes(t)}
	before := readNote(t, m, "compras.md")
	press(m, "2")
	m.tasks.list.set(taskIndex(t, m, "Pan"), len(m.c.tasks))

	ctrlV(m)

	lines := strings.Split(readNote(t, m, "compras.md"), "\n")
	orig := strings.Split(before, "\n")
	pan := -1
	for i, l := range lines {
		if strings.Contains(l, "Pan") {
			pan = i
		}
	}
	if pan < 0 || lines[pan+1] != "" || !strings.HasPrefix(lines[pan+2], "![](assets/compras-") {
		t.Fatalf("la imagen no quedó debajo de 'Pan':\n%s", strings.Join(lines, "\n"))
	}
	rest := append(append([]string{}, lines[:pan+1]...), lines[pan+3:]...)
	if strings.Join(rest, "\n") != strings.Join(orig, "\n") {
		t.Errorf("cambió algo más que la inserción:\n%q\nvs\n%q", rest, orig)
	}
	if cur := m.tasks.current(); cur == nil || !strings.Contains(cur.Text, "Pan") {
		t.Errorf("el cursor ya no está sobre la tarea: %+v", cur)
	}
}

// TestCtrlVWithoutImage: si el portapapeles no tiene imagen no se toca la nota ni
// queda nada a medias en assets/.
func TestCtrlVWithoutImage(t *testing.T) {
	m := newTestModel(t, 120, 35)
	clip := &fakeClip{err: errors.New("sin imagen")}
	m.c.clip.Reader = clip
	base := m.c.store.BaseDir
	before := readNote(t, m, "compras.md")
	m.notes.selectPath(filepath.Join(base, "compras.md"))
	nAssets := len(assetFiles(t, base))

	ctrlV(m)

	if clip.calls != 1 || readNote(t, m, "compras.md") != before || len(assetFiles(t, base)) != nAssets {
		t.Errorf("un fallo no debe cambiar nada (llamadas=%d)", clip.calls)
	}
	if !strings.Contains(m.c.status, "No se pudo guardar la imagen") {
		t.Errorf("el estado no avisa del error: %q", m.c.status)
	}
}

// TestCtrlVOnFolderDoesNothing: sin una nota a la vista no se lee el portapapeles.
func TestCtrlVOnFolderDoesNothing(t *testing.T) {
	m := newTestModel(t, 120, 35)
	clip := &fakeClip{data: pngBytes(t)}
	m.c.clip.Reader = clip
	m.notes.selectPath(filepath.Join(m.c.store.BaseDir, "proyectos"))
	ctrlV(m)
	if clip.calls != 0 || !strings.Contains(m.c.status, "Abre una nota") {
		t.Errorf("sobre una carpeta no debe pegarse (llamadas=%d, estado %q)", clip.calls, m.c.status)
	}
}

// TestCtrlVStaleNote (X10): si la nota cambió por fuera, no se pisa y la imagen
// guardada no queda huérfana.
func TestCtrlVStaleNote(t *testing.T) {
	m := newTestModel(t, 120, 35)
	m.c.clip.Reader = &fakeClip{data: pngBytes(t)}
	base := m.c.store.BaseDir
	path := filepath.Join(base, "compras.md")
	m.notes.selectPath(path)
	nAssets := len(assetFiles(t, base))

	external := readNote(t, m, "compras.md") + "\nescrito por otro editor\n"
	if err := os.WriteFile(path, []byte(external), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	_ = os.Chtimes(path, later, later)

	ctrlV(m)

	if readNote(t, m, "compras.md") != external {
		t.Fatal("se pisó la edición externa")
	}
	if len(assetFiles(t, base)) != nAssets {
		t.Error("quedó una imagen huérfana en assets/")
	}
	if !strings.Contains(m.c.status, "cambió por fuera") {
		t.Errorf("no avisó: %q", m.c.status)
	}
	// ya recargada, el siguiente intento funciona y conserva la edición externa
	ctrlV(m)
	if got := readNote(t, m, "compras.md"); !strings.HasPrefix(got, external) || !strings.Contains(got, "![](assets/compras-") {
		t.Errorf("el reintento debía agregar la imagen conservando lo externo:\n%s", got)
	}
}

// TestCmdVFilePath (H3-2): Cmd+V con un archivo de imagen copiado llega como su
// ruta en un pegado; se copia a assets/ y el original no se toca.
func TestCmdVFilePath(t *testing.T) {
	m := newTestModel(t, 120, 35)
	base := m.c.store.BaseDir
	ext := t.TempDir()
	src := filepath.Join(ext, "mi captura (1).PNG")
	if err := os.WriteFile(src, pngBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	m.notes.selectPath(filepath.Join(base, "compras.md"))
	before := readNote(t, m, "compras.md")

	// macOS/Linux escapan con "\"; Windows Terminal entrega la ruta entre comillas
	pasted := strings.NewReplacer(" ", `\ `, "(", `\(`, ")", `\)`).Replace(src)
	if runtime.GOOS == "windows" {
		pasted = `"` + src + `"`
	}
	m.Update(tea.PasteMsg{Content: pasted})

	after := readNote(t, m, "compras.md")
	if !strings.HasPrefix(after, before+"\n![](assets/compras-") || !strings.HasSuffix(after, ".png)\n") {
		t.Fatalf("la nota debía terminar con la referencia a la copia:\n%s", after)
	}
	if got, _ := os.ReadFile(src); string(got) != string(pngBytes(t)) {
		t.Error("el archivo original cambió")
	}
	if len(assetFiles(t, base)) < 1 {
		t.Error("no se copió a assets/")
	}

	// varias rutas: una referencia por imagen, cada una con su propia comprobación de mtime
	src2 := filepath.Join(ext, "otra.jpg")
	_ = os.WriteFile(src2, pngBytes(t), 0o644)
	before = readNote(t, m, "compras.md")
	m.Update(tea.PasteMsg{Content: src + "\n" + src2 + "\n"})
	if n := strings.Count(readNote(t, m, "compras.md"), "![](assets/") - strings.Count(before, "![](assets/"); n != 2 {
		t.Errorf("se esperaban 2 referencias nuevas, hubo %d", n)
	}
}

// TestPasteTextIsIgnored: lo pegado que no es una imagen no cambia la nota y avisa.
func TestPasteTextIsIgnored(t *testing.T) {
	m := newTestModel(t, 120, 35)
	base := m.c.store.BaseDir
	m.notes.selectPath(filepath.Join(base, "compras.md"))
	before := readNote(t, m, "compras.md")
	nAssets := len(assetFiles(t, base))
	txt := filepath.Join(t.TempDir(), "x.txt")
	_ = os.WriteFile(txt, []byte("x"), 0o644)
	for _, c := range []string{"hola mundo", txt, "", filepath.Join(t.TempDir(), "no-existe.png")} {
		m.Update(tea.PasteMsg{Content: c})
	}
	if readNote(t, m, "compras.md") != before || len(assetFiles(t, base)) != nAssets {
		t.Error("un pegado que no es imagen no debe cambiar nada")
	}
	if !strings.Contains(m.c.status, "no es una imagen") {
		t.Errorf("no avisó: %q", m.c.status)
	}
}

// TestPasteIntoNamePopup: con el popup de nombre abierto, el pegado va al campo
// de texto y no se interpreta como una imagen.
func TestPasteIntoNamePopup(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "c")
	ip, ok := m.c.top().(*inputPopup)
	if !ok {
		t.Fatal("c no abrió el popup de nombre")
	}
	m.Update(tea.PasteMsg{Content: " pegado"})
	if !strings.HasSuffix(ip.input.Value(), " pegado") {
		t.Errorf("el campo no recibió el texto pegado: %q", ip.input.Value())
	}
}

// TestCtrlVInCheatsheet (X9): el atajo figura en el cheatsheet porque está en el keymap.
func TestCtrlVInCheatsheet(t *testing.T) {
	m := newTestModel(t, 120, 35)
	press(m, "?")
	if out := plain(m); !strings.Contains(out, "ctrl+v") || !strings.Contains(out, "Pegar imagen") {
		t.Errorf("el cheatsheet no lista Ctrl+V:\n%s", out)
	}
}
