package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// linksModel arma una carpeta con notas que se enlazan y selecciona "hub".
func linksModel(t *testing.T) (*AppModel, string) {
	t.Helper()
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		os.Remove(n.Path)
	}
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	w := func(rel, body string) { os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644) }
	w("hub.md", "# Hub\n\nVer [[destino]] y [[destino|el alias]] y [[no-existe]] y [[sub/otra#Sección]].\n\n```\n[[destino]] en código\n```\n\nFin del hub.\n")
	w("destino.md", "# Destino\n\ntexto del destino\n")
	w("sub/otra.md", "# Otra\n\n## Sección\n\nlínea con la sección\n")
	w("apunta.md", "# Apunta\n\nlínea uno\nmira [[hub]] aquí\n")
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(filepath.Join(dir, "hub.md"))
	return m, dir
}

func rawView(m *AppModel) string { return m.View().Content }

// TestWikilinksAreHighlighted (C.2): los [[enlaces]] se ven con su texto (el alias si lo hay) y sin corchetes ni marcadores, en azul
// subrayado si la nota existe y en durazno si no; los que están en código se quedan como estaban.
func TestWikilinksAreHighlighted(t *testing.T) {
	m, _ := linksModel(t)
	out := plain(m)
	for _, want := range []string{"Ver destino y el alias y no-existe y sub/otra > Sección.", "[[destino]] en código", "Fin del hub."} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.ContainsAny(out, string(linkOpen)+string(linkClose)) || strings.Contains(out, "[[no-existe]]") {
		t.Errorf("no deben verse marcadores ni corchetes de enlaces:\n%s", out)
	}
	raw := rawView(m)
	if !strings.Contains(raw, "\x1b[4") && !strings.Contains(raw, ";4;") && !strings.Contains(raw, ";4m") {
		t.Errorf("los enlaces van subrayados")
	}
	if len(m.preview.links) != 4+1 { // 4 enlaces en el texto y 1 backlink (apunta.md)
		t.Fatalf("enlaces de la vista previa: %d (%+v)", len(m.preview.links), m.preview.links)
	}
	if !m.preview.links[2].missing() || m.preview.links[0].missing() || m.preview.links[3].Path == "" {
		t.Errorf("resolución: %+v", m.preview.links)
	}
	// los demás consumidores de las líneas no ven marcadores
	for _, l := range m.preview.lines(m.previewNote(), 80) {
		if strings.ContainsAny(l, string(linkOpen)+string(linkClose)) {
			t.Fatalf("lines() no debe devolver marcadores: %q", l)
		}
	}
}

// TestWikilinksFollowWithKeysAndMouse (C.2): con la vista previa enfocada, n y N recorren los enlaces; Enter sigue el seleccionado
// (va a la nota); Esc lo suelta; un clic sobre un enlace lo sigue; un enlace con # lleva a su encabezado.
func TestWikilinksFollowWithKeysAndMouse(t *testing.T) {
	m, dir := linksModel(t)
	press(m, "4") // enfoca la vista previa
	if m.focus != panelPreview {
		t.Fatalf("foco = %v", m.focus)
	}
	press(m, "n")
	if m.preview.sel != 1 {
		t.Fatalf("n selecciona el primer enlace: %d", m.preview.sel)
	}
	press(m, "n") // el segundo
	if m.preview.sel != 2 {
		t.Errorf("sel = %d", m.preview.sel)
	}
	press(m, "N", "N") // vuelve al primero y da la vuelta al último (el backlink)
	if m.preview.sel != len(m.preview.links) {
		t.Errorf("N da la vuelta: %d de %d", m.preview.sel, len(m.preview.links))
	}
	press(m, "esc")
	if m.preview.sel != 0 || m.focus != panelPreview {
		t.Errorf("Esc suelta el enlace sin salir del panel: sel=%d foco=%v", m.preview.sel, m.focus)
	}
	press(m, "n", "enter") // el primero: [[destino]]
	if n := m.previewNote(); n == nil || n.Path != filepath.Join(dir, "destino.md") {
		t.Fatalf("Enter debía llevar a destino.md: %+v", m.previewNote())
	}
	// volver al hub y seguir [[sub/otra#Sección]] con un clic
	m.notes.selectPath(filepath.Join(dir, "hub.md"))
	m.preview.reset()
	m.afterChange()
	x, y, ok := cellOf(m, "sub/otra > Sección")
	if !ok {
		t.Fatalf("no se ve el enlace:\n%s", plain(m))
	}
	m.Update(tea.MouseClickMsg{X: x + 1, Y: y, Button: tea.MouseLeft})
	if n := m.previewNote(); n == nil || n.Path != filepath.Join(dir, "sub", "otra.md") {
		t.Fatalf("el clic debía llevar a sub/otra.md: %+v", m.previewNote())
	}
	if !strings.Contains(lastRow(m), "otra.md:3") {
		t.Errorf("con el # va a la línea del encabezado: %q", lastRow(m))
	}
	// un clic fuera de un enlace no hace nada más que enfocar
	m.notes.selectPath(filepath.Join(dir, "hub.md"))
	m.afterChange()
	before := m.previewNote().Path
	x, y, _ = cellOf(m, "Fin del hub")
	m.Update(tea.MouseClickMsg{X: x + 2, Y: y, Button: tea.MouseLeft})
	if m.previewNote().Path != before {
		t.Error("un clic fuera de un enlace no debe moverse")
	}
}

// TestMissingWikilinkOffersToCreate (C.2): seguir un enlace a una nota inexistente ofrece crearla (y la crea junto a la que lo contiene).
func TestMissingWikilinkOffersToCreate(t *testing.T) {
	m, dir := linksModel(t)
	press(m, "4", "n", "n", "n") // el tercero: [[no-existe]]
	if !m.preview.links[m.preview.sel-1].missing() {
		t.Fatalf("el tercero debía ser el roto: %+v", m.preview.links[m.preview.sel-1])
	}
	press(m, "enter")
	if out := plain(m); !strings.Contains(out, "no-existe") || !strings.Contains(out, "¿Quieres crearla?") {
		t.Fatalf("debía ofrecer crearla:\n%s", out)
	}
	press(m, "n") // no
	if _, err := os.Stat(filepath.Join(dir, "no-existe.md")); err == nil {
		t.Fatal("con n no se crea")
	}
	press(m, "enter")
	press(m, "y")
	if _, err := os.Stat(filepath.Join(dir, "no-existe.md")); err != nil {
		t.Fatalf("con y se crea la nota: %v", err)
	}
	if n := m.previewNote(); n == nil || n.Path != filepath.Join(dir, "no-existe.md") {
		t.Errorf("tras crearla se va a ella: %+v", m.previewNote())
	}
}

// TestBacklinksSection (C.2): la vista previa termina con las notas que enlazan a la actual (con su línea) y se siguen como cualquier enlace.
func TestBacklinksSection(t *testing.T) {
	m, dir := linksModel(t)
	m.notes.selectPath(filepath.Join(dir, "destino.md"))
	m.afterChange()
	out := plain(m)
	if !strings.Contains(out, "Enlaces a esta nota (1)") || !strings.Contains(out, "hub:3") || !strings.Contains(out, "Ver destino y el alias y no-existe") || strings.Contains(out, "[[destino]] y") {
		t.Errorf("debía listar la línea de hub con sus enlaces (una entrada por línea; el de código no cuenta; sin corchetes):\n%s", out)
	}
	if strings.Contains(out, "hub:6") {
		t.Errorf("el enlace dentro de código no es un backlink:\n%s", out)
	}
	// una nota sin enlaces entrantes no tiene la sección (nadie enlaza a apunta.md; ella enlaza a hub)
	m.notes.selectPath(filepath.Join(dir, "apunta.md"))
	m.afterChange()
	if strings.Contains(plain(m), "Enlaces a esta nota") {
		t.Errorf("apunta.md no tiene enlaces entrantes:\n%s", plain(m))
	}
	m.notes.selectPath(filepath.Join(dir, "sub", "otra.md"))
	m.afterChange()
	if strings.Contains(plain(m), "Enlaces a esta nota (1)") == false {
		t.Errorf("otra.md tiene un backlink:\n%s", plain(m))
	}
	// seguir un backlink: va a la nota que enlaza, a la línea del enlace
	m.notes.selectPath(filepath.Join(dir, "hub.md"))
	m.afterChange()
	press(m, "4", "N") // el último enlace de la vista previa: el backlink de apunta.md
	if l := m.preview.links[m.preview.sel-1]; !l.Back || l.Line != 4 {
		t.Fatalf("el último debía ser el backlink de apunta.md línea 4: %+v", l)
	}
	press(m, "enter")
	if n := m.previewNote(); n == nil || n.Path != filepath.Join(dir, "apunta.md") {
		t.Errorf("seguir el backlink lleva a apunta.md: %+v", m.previewNote())
	}
}

func fileText(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestRenameOffersToUpdateLinks (C.2): renombrar una nota a la que otras enlazan muestra las líneas que cambian (antes y después) y deja
// actualizarlas (y), renombrar sin tocarlas (n) o cancelar (Esc); solo se reescriben esas líneas.
func TestRenameOffersToUpdateLinks(t *testing.T) {
	m, dir := linksModel(t)
	hub := filepath.Join(dir, "hub.md")
	rename := func(to string) {
		press(m, "1", "r")
		for i := 0; i < 20; i++ {
			press(m, "backspace")
		}
		for _, r := range to {
			press(m, string(r))
		}
		press(m, "enter")
	}
	m.notes.selectPath(filepath.Join(dir, "destino.md"))
	m.afterChange()
	beforeHub := fileText(t, hub)

	// Esc cancela todo
	rename("nuevo")
	out := plain(m)
	for _, want := range []string{"1 línea(s) de 1 nota(s)", "hub.md:3", "- Ver [[destino]] y [[destino|el alias]]", "+ Ver [[nuevo]] y [[nuevo|el alias]]", "[y] Actualizar los enlaces"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	press(m, "esc")
	if _, err := os.Stat(filepath.Join(dir, "destino.md")); err != nil || fileText(t, hub) != beforeHub {
		t.Fatal("Esc no debe renombrar ni tocar nada")
	}

	// n: renombra sin tocar los enlaces
	rename("nuevo")
	press(m, "n")
	if _, err := os.Stat(filepath.Join(dir, "nuevo.md")); err != nil || fileText(t, hub) != beforeHub {
		t.Fatalf("con n solo se renombra: %v", err)
	}
	os.Rename(filepath.Join(dir, "nuevo.md"), filepath.Join(dir, "destino.md"))
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(filepath.Join(dir, "destino.md"))

	// y: renombra y reescribe solo esas líneas
	rename("nuevo")
	press(m, "y")
	if _, err := os.Stat(filepath.Join(dir, "nuevo.md")); err != nil {
		t.Fatalf("con y se renombra: %v", err)
	}
	want := strings.Replace(beforeHub, "Ver [[destino]] y [[destino|el alias]]", "Ver [[nuevo]] y [[nuevo|el alias]]", 1)
	if got := fileText(t, hub); got != want {
		t.Errorf("hub.md tras actualizar:\n%s\nse esperaba:\n%s", got, want)
	}
	if !strings.Contains(lastRow(m), "1 línea(s) con enlaces actualizada(s)") {
		t.Errorf("el aviso cuenta lo hecho: %q", lastRow(m))
	}
	// la línea del enlace dentro de código no se tocó
	if !strings.Contains(fileText(t, hub), "[[destino]] en código") {
		t.Error("el código no se reescribe")
	}

	// una nota sin enlaces entrantes se renombra sin preguntar
	m.notes.selectPath(filepath.Join(dir, "sub", "otra.md"))
	m.afterChange()
	// (otra.md sí tiene un enlace desde hub: se usa apunta.md, que nadie enlaza ... hub enlaza a hub? no; apunta.md no tiene entrantes)
	m.notes.selectPath(filepath.Join(dir, "apunta.md"))
	m.afterChange()
	rename("apuntada")
	if _, err := os.Stat(filepath.Join(dir, "apuntada.md")); err != nil {
		t.Errorf("sin enlaces entrantes se renombra directo: %v", err)
	}
	_ = ansi.Strip
}
