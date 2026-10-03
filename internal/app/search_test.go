package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/search"
)

// drive ejecuta un comando de Bubble Tea y entrega al modelo los mensajes que produce (también los de un Batch), hasta que no haya más.
func drive(m *AppModel, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			drive(m, c)
		}
	default:
		_, next := m.Update(msg)
		drive(m, next)
	}
}

// typeSearch escribe texto en el popup de búsqueda y deja correr lo que se lanza (la pausa y la búsqueda).
func typeSearch(m *AppModel, text string) {
	for _, r := range text {
		_, cmd := m.Update(keyMsg(string(r)))
		if cmd != nil {
			defer drive(m, cmd) // solo la búsqueda de la última tecla importa: las anteriores se cancelan
		}
	}
}

// searchModel arma una carpeta con notas conocidas, dentro de carpetas, y abre la búsqueda.
func searchModel(t *testing.T) (*AppModel, string) {
	t.Helper()
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	os.MkdirAll(filepath.Join(dir, "proyectos", "web"), 0o755)
	var long strings.Builder
	long.WriteString("# Largo\n\n")
	for i := 1; i <= 80; i++ {
		long.WriteString("línea de relleno número " + strings.Repeat("x", i%7) + "\n\n")
	}
	long.WriteString("aquí está la AGUJA_UNICA que se busca\n\n")
	long.WriteString("y más texto después\n")
	os.WriteFile(filepath.Join(dir, "proyectos", "web", "largo.md"), []byte(long.String()), 0o644)
	os.WriteFile(filepath.Join(dir, "uno.md"), []byte("# Uno\n\nel zorzalino canta\n- [ ] comprar zorzalino\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "dos.md"), []byte("# Dos\n\nCANCIÓN de cuna\n"), 0o644)
	m.c.reload()
	m.afterChange()
	press(m, "/")
	return m, dir
}

// TestSearchPopupLiveResults (C.1): `/` abre la búsqueda; al escribir salen los resultados (nota:línea, contexto con lo hallado)
// y cada texto nuevo reemplaza a la búsqueda anterior.
func TestSearchPopupLiveResults(t *testing.T) {
	m, _ := searchModel(t)
	sp, ok := m.c.top().(*searchPopup)
	if !ok {
		t.Fatalf("`/` debía abrir la búsqueda: %T", m.c.top())
	}
	if out := plain(m); !strings.Contains(out, "Buscar") || !strings.Contains(out, "Escribe para buscar") {
		t.Errorf("estado inicial:\n%s", out)
	}
	typeSearch(m, "zorzalino")
	out := plain(m)
	for _, want := range []string{"2 resultado(s) en 3 nota(s)", "uno.md:3", "el zorzalino canta", "uno.md:4", "comprar zorzalino"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if sp.n != 2 {
		t.Errorf("resultados = %d", sp.n)
	}
	press(m, "esc")
	press(m, "/")
	typeSearch(m, "canción")
	if out := plain(m); !strings.Contains(out, "dos.md:3") || !strings.Contains(out, "CANCIÓN de cuna") {
		t.Errorf("mayúsculas y acentos:\n%s", out)
	}
	typeSearch(m, "!!!") // ahora la consulta es "canción!!!": sin resultados
	if out := plain(m); !strings.Contains(out, "Sin resultados") {
		t.Errorf("sin resultados:\n%s", out)
	}
}

// TestSearchIgnoresStaleResultsAndCancels (C.1): los resultados de un texto anterior no pisan a los del actual, y cerrar la búsqueda
// cancela lo que estaba corriendo.
func TestSearchIgnoresStaleResultsAndCancels(t *testing.T) {
	m, _ := searchModel(t)
	sp := m.c.top().(*searchPopup)
	typeSearch(m, "zorzalino")
	stale := searchDoneMsg{gen: sp.gen - 1, query: "viejo", res: search.Result{Matches: []search.Match{{Rel: "x.md", Line: 1, Text: "x"}}}}
	m.Update(stale)
	if sp.n != 2 {
		t.Errorf("un resultado de una generación anterior no debe pisar el actual: n=%d", sp.n)
	}
	// un tick viejo no lanza nada
	if cmd := sp.run(sp.gen - 1); cmd != nil {
		t.Error("un tick de una generación anterior no debe buscar")
	}
	// cancelación: la búsqueda pendiente se cancela al cerrar
	_, cmd := m.Update(keyMsg("a"))
	_ = cmd
	gen := sp.gen
	press(m, "esc")
	if sp.gen == gen {
		t.Error("cerrar la búsqueda debe invalidar la generación (cancelar lo pendiente)")
	}
	if m.c.top() != nil {
		t.Error("Esc cierra el popup")
	}
	if run := sp.run(gen); run != nil {
		t.Error("tras cerrar no se lanza ninguna búsqueda")
	}
}

// TestSearchDoesNotBlockNavigation (C.1): mientras se escribe, las teclas se procesan sin esperar a la búsqueda (se lanza en un
// comando aparte) y los resultados no llegan por el camino síncrono.
func TestSearchDoesNotBlockNavigation(t *testing.T) {
	m, dir := searchModel(t)
	for i := 0; i < 2000; i++ { // muchas notas para que una búsqueda cueste
		os.WriteFile(filepath.Join(dir, "relleno-"+strings.Repeat("a", i%5)+string(rune('a'+i%26))+string(rune('a'+i/26%26))+string(rune('a'+i/676))+".md"), []byte(strings.Repeat("texto de relleno con zorzalino\n", 200)), 0o644)
	}
	start := time.Now()
	for _, r := range "zorzalino" {
		if _, cmd := m.Update(keyMsg(string(r))); cmd == nil {
			continue
		}
	}
	if d := time.Since(start); d > 150*time.Millisecond {
		t.Errorf("escribir 9 teclas tardó %v: la búsqueda no debe correr dentro de Update", d)
	}
	if sp := m.c.top().(*searchPopup); sp.n != 0 {
		t.Error("los resultados no pueden estar antes de que corra el comando")
	}
}

// TestSearchJumpsToNoteAndLine (C.1): Enter lleva a la nota (abre sus carpetas) y a la línea: la vista previa queda enfocada y la
// línea hallada a la vista; el segundo clic en un resultado hace lo mismo.
func TestSearchJumpsToNoteAndLine(t *testing.T) {
	m, dir := searchModel(t)
	typeSearch(m, "AGUJA_UNICA")
	if out := plain(m); !strings.Contains(out, "proyectos/web/largo.md:") {
		t.Fatalf("falta el resultado:\n%s", out)
	}
	_, cmd := m.Update(keyMsg("enter"))
	drive(m, cmd)
	if m.c.top() != nil {
		t.Fatal("Enter cierra la búsqueda")
	}
	want := filepath.Join(dir, "proyectos", "web", "largo.md")
	if n := m.previewNote(); n == nil || n.Path != want {
		t.Fatalf("la nota seleccionada debía ser %s: %+v", want, n)
	}
	if m.focus != panelPreview {
		t.Errorf("el foco debía quedar en la vista previa: %v", m.focus)
	}
	if !strings.Contains(plain(m), "AGUJA_UNICA") {
		t.Errorf("la línea hallada debe estar a la vista (scrollY=%d):\n%s", m.preview.scrollY, plain(m))
	}
	if m.preview.scrollY == 0 {
		t.Error("la vista previa debía desplazarse hasta la línea")
	}
	if !strings.Contains(lastRow(m), "largo.md:") {
		t.Errorf("el aviso dice a dónde se fue: %q", lastRow(m))
	}
	// segundo clic en un resultado
	press(m, "/")
	typeSearch(m, "zorzalino")
	x, y, ok := cellOf(m, "uno.md:4") // el segundo resultado: el cursor empieza en el primero
	if !ok {
		t.Fatalf("no se ve el resultado:\n%s", plain(m))
	}
	_, c1 := m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	drive(m, c1)
	if sp, ok := m.c.top().(*searchPopup); !ok || sp.list.cursor != 1 {
		t.Fatal("el primer clic solo selecciona el segundo resultado")
	}
	_, c2 := m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	drive(m, c2)
	if sp := m.c.top(); sp != nil || m.previewNote() == nil || filepath.Base(m.previewNote().Path) != "uno.md" {
		t.Errorf("el segundo clic debía saltar a uno.md: %v", m.previewNote())
	}
}
