package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func tplModel(t *testing.T) (*AppModel, string) {
	t.Helper()
	m := newTestModel(t, 120, 35)
	dir := m.c.store.BaseDir
	os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	os.WriteFile(filepath.Join(dir, "templates", "daily.md"), []byte("# Diario {{date}}\n\n## Hoy\n- [ ] revisar el día\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "templates", "reunion.md"), []byte("# {{title}}\n\nFecha: {{date}} {{time}}\n- [ ] orden del día\n"), 0o644)
	m.c.reload()
	m.afterChange()
	return m, dir
}

// TestDailyNoteShortcut (C.5): T crea journal/AAAA-MM-DD.md con la plantilla daily, lo abre en la vista previa y avisa; la segunda vez
// lo abre sin tocarlo.
func TestDailyNoteShortcut(t *testing.T) {
	m, dir := tplModel(t)
	today := time.Now().Format("2006-01-02")
	path := filepath.Join(dir, "journal", today+".md")
	press(m, "T")
	if b, err := os.ReadFile(path); err != nil || string(b) != "# Diario "+today+"\n\n## Hoy\n- [ ] revisar el día\n" {
		t.Fatalf("la nota diaria: %v %q", err, b)
	}
	if n := m.previewNote(); n == nil || n.Path != path {
		t.Errorf("T debe dejar la nota diaria en la vista previa: %+v", m.previewNote())
	}
	if !strings.Contains(lastRow(m), "Nota diaria creada") {
		t.Errorf("aviso: %q", lastRow(m))
	}
	os.WriteFile(path, []byte("# mía\n"), 0o644)
	m.c.reload()
	press(m, "T")
	if b, _ := os.ReadFile(path); string(b) != "# mía\n" {
		t.Errorf("la segunda vez no debe tocarla: %q", b)
	}
	if !strings.Contains(lastRow(m), "Nota diaria:") || strings.Contains(lastRow(m), "creada") {
		t.Errorf("aviso de la segunda vez: %q", lastRow(m))
	}
	// desde el Kanban también vuelve a las notas
	press(m, "W")
	press(m, "T")
	if m.kanbanOn {
		t.Error("T desde el Kanban abre la nota: sale del tablero")
	}
}

// TestNewNoteFromTemplateShortcut (C.5): C lista las plantillas de templates/, pide el nombre y crea la nota con las variables
// reemplazadas; sin plantillas avisa; las casillas de una plantilla no salen en Tareas.
func TestNewNoteFromTemplateShortcut(t *testing.T) {
	m, dir := tplModel(t)
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "C")
	out := plain(m)
	if !strings.Contains(out, "daily") || !strings.Contains(out, "reunion") {
		t.Fatalf("el popup debe listar las plantillas:\n%s", out)
	}
	press(m, "down", "enter") // reunion (daily, reunion)
	press(m, "ctrl+a", "ctrl+k")
	for _, r := range "Equipo semanal" {
		press(m, string(r))
	}
	press(m, "enter")
	b, err := os.ReadFile(filepath.Join(dir, "equipo-semanal.md"))
	if err != nil {
		t.Fatalf("no creó la nota: %v", err)
	}
	if !strings.HasPrefix(string(b), "# Equipo semanal\n\nFecha: "+time.Now().Format("2006-01-02")+" ") {
		t.Errorf("contenido: %q", b)
	}
	for _, tk := range m.c.tasks {
		if strings.Contains(tk.NotePath, "templates") {
			t.Errorf("una tarea de una plantilla salió en Tareas: %+v", tk)
		}
	}
	// sin plantillas
	os.RemoveAll(filepath.Join(dir, "templates"))
	m.c.reload()
	press(m, "C")
	if !strings.Contains(lastRow(m), "No hay plantillas") {
		t.Errorf("sin plantillas debe avisar: %q", lastRow(m))
	}
}

// TestTemplateProblemsInTheTUI (ORD-014 M1 y M2): C con una plantilla binaria no crea la nota y avisa; con una variable desconocida la crea y
// avisa "Variable desconocida: {{…}}"; T con una daily binaria no crea journal/.
func TestTemplateProblemsInTheTUI(t *testing.T) {
	m, dir := tplModel(t)
	os.WriteFile(filepath.Join(dir, "templates", "binaria.md"), []byte("# x\x00\xff"), 0o644)
	os.WriteFile(filepath.Join(dir, "templates", "vars.md"), []byte("# {{title}}\n{{fecha}}\n"), 0o644)
	os.Remove(filepath.Join(dir, "templates", "daily.md"))
	m.c.reload()
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	create := func(tpl, name string) {
		press(m, "C")
		pickTemplate(m, tpl)
		press(m, "ctrl+a", "ctrl+k")
		for _, r := range name {
			press(m, string(r))
		}
		press(m, "enter")
	}
	create("binaria", "Rota")
	if _, err := os.Stat(filepath.Join(dir, "rota.md")); err == nil {
		t.Error("una plantilla binaria no crea la nota")
	}
	if !strings.Contains(lastRow(m), "plantilla") && !strings.Contains(lastRow(m), "UTF-8") {
		t.Errorf("debe avisar de que la plantilla no vale: %q", lastRow(m))
	}
	create("vars", "Con var")
	if b, err := os.ReadFile(filepath.Join(dir, "con-var.md")); err != nil || !strings.Contains(string(b), "{{fecha}}") {
		t.Fatalf("la nota con variable desconocida se crea: %v %q", err, b)
	}
	if !strings.Contains(lastRow(m), "Variable desconocida: {{fecha}}") {
		t.Errorf("aviso: %q", lastRow(m))
	}
	os.WriteFile(filepath.Join(dir, "templates", "daily.md"), []byte("# x\x00"), 0o644)
	press(m, "T")
	if _, err := os.Stat(filepath.Join(dir, "journal")); err == nil {
		if es, _ := os.ReadDir(filepath.Join(dir, "journal")); len(es) != 0 {
			t.Errorf("T con una daily binaria no debe crear la nota: %v", es)
		}
	}
}

// pickTemplate mueve el cursor del popup de plantillas a tpl y lo confirma.
func pickTemplate(m *AppModel, tpl string) {
	p, ok := m.c.top().(*templatePopup)
	if !ok {
		return
	}
	for i, n := range p.names {
		if n == tpl {
			p.list.cursor = i
		}
	}
	press(m, "enter")
}

// TestNewFromTemplateNeverLandsInTemplates (ORD-016 C.4 / L6): con la carpeta templates/ (o una nota de dentro) seleccionada, C no crea la nota nueva
// dentro de templates/ —donde pasaría a ser una plantilla y dejaría de contar sus tareas— sino en la raíz de la carpeta de notas. Con otra carpeta
// seleccionada sigue creando ahí, y c (nota normal) en templates/ sigue creando una plantilla a propósito.
func TestNewFromTemplateNeverLandsInTemplates(t *testing.T) {
	m, dir := tplModel(t)
	create := func(name string) {
		press(m, "C")
		pickTemplate(m, "reunion")
		press(m, "ctrl+a", "ctrl+k")
		for _, r := range name {
			press(m, string(r))
		}
		press(m, "enter")
	}
	exists := func(rel string) bool { _, err := os.Stat(filepath.Join(dir, rel)); return err == nil }
	// la carpeta templates seleccionada
	m.notes.selectPath(filepath.Join(dir, "templates"))
	create("Desde carpeta")
	if exists("templates/desde-carpeta.md") || !exists("desde-carpeta.md") {
		t.Errorf("con la carpeta templates seleccionada la nota va a la raíz: en templates=%v en raíz=%v", exists("templates/desde-carpeta.md"), exists("desde-carpeta.md"))
	}
	// una plantilla seleccionada
	m.notes.selectPath(filepath.Join(dir, "templates", "daily.md"))
	create("Desde plantilla")
	if exists("templates/desde-plantilla.md") || !exists("desde-plantilla.md") {
		t.Errorf("con una plantilla seleccionada también va a la raíz")
	}
	// otra carpeta: sigue ahí
	os.MkdirAll(filepath.Join(dir, "proyecto"), 0o755)
	m.c.reload()
	m.afterChange()
	m.notes.selectPath(filepath.Join(dir, "proyecto"))
	create("En proyecto")
	if !exists("proyecto/en-proyecto.md") {
		t.Error("con otra carpeta seleccionada la nota se crea ahí")
	}
	// la nota creada no es una plantilla: sus tareas cuentan
	found := false
	for _, tk := range m.c.tasks {
		if tk.Text == "orden del día" && strings.Contains(tk.NotePath, "desde-carpeta") {
			found = true
		}
	}
	if !found {
		t.Error("las tareas de la nota nueva aparecen en Tareas (no es una plantilla)")
	}
	// c normal en templates/ crea una plantilla a propósito
	m.notes.selectPath(filepath.Join(dir, "templates"))
	press(m, "c")
	press(m, "ctrl+a", "ctrl+k")
	for _, r := range "Mi molde" {
		press(m, string(r))
	}
	press(m, "enter")
	if !exists("templates/mi-molde.md") {
		t.Error("c dentro de templates/ crea una plantilla a propósito")
	}
}

// TestNewNoteSuggestedNameIsReplacedByTyping (ORD-017 F7 / L10): el nombre sugerido ("Nueva nota 4") se reemplaza al escribir (antes quedaba "nueva-nota-4dentro.md");
// Backspace o las flechas lo dejan editable, y Enter sin escribir lo acepta. Vale para c, C y la carpeta nueva; renombrar sigue editando el nombre actual.
func TestNewNoteSuggestedNameIsReplacedByTyping(t *testing.T) {
	m, dir := tplModel(t)
	exists := func(rel string) bool { _, err := os.Stat(filepath.Join(dir, rel)); return err == nil }
	typeRaw := func(s string) {
		for _, r := range s {
			press(m, string(r))
		}
	}
	// c: escribir reemplaza la sugerencia
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "c")
	typeRaw("Dentro")
	press(m, "enter")
	if !exists("dentro.md") {
		entries, _ := os.ReadDir(dir)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("c: escribir reemplaza la sugerencia; archivos: %v", names)
	}
	// C: igual
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "C")
	pickTemplate(m, "reunion")
	typeRaw("Sync")
	press(m, "enter")
	if !exists("sync.md") {
		t.Error("C: escribir reemplaza la sugerencia")
	}
	// Backspace deja editar la sugerencia (quita su último carácter y lo escrito se agrega)
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "c")
	press(m, "backspace")
	typeRaw("ab")
	press(m, "enter")
	found := false
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "nueva-nota-") && strings.HasSuffix(e.Name(), "ab.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("Backspace deja editar la sugerencia (nueva-nota-…ab.md): %v", entries)
	}
	// una flecha también la deja editable
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "c")
	press(m, "left")
	typeRaw("Z")
	press(m, "enter")
	found = false
	entries, _ = os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "nueva-nota-") && strings.Contains(e.Name(), "z") && !strings.HasSuffix(e.Name(), "ab.md") {
			found = true
		}
	}
	if !found {
		t.Errorf("las flechas dejan editar la sugerencia: %v", entries)
	}
	// Enter sin escribir acepta la sugerencia
	before := len(entries)
	m.notes.selectPath(filepath.Join(dir, "compras.md"))
	press(m, "c", "enter")
	if after, _ := os.ReadDir(dir); len(after) != before+1 {
		t.Error("Enter acepta la sugerencia")
	}
	// carpeta nueva: igual
	press(m, "F")
	typeRaw("viajes")
	press(m, "enter")
	if !exists("viajes") {
		t.Error("F: escribir reemplaza el nombre sugerido de la carpeta")
	}
	// renombrar: sigue editando el nombre actual (no se reemplaza)
	m.notes.selectPath(filepath.Join(dir, "dentro.md"))
	press(m, "r")
	typeRaw("2")
	press(m, "enter")
	if !exists("dentro2.md") {
		t.Error("renombrar edita el nombre actual: escribir lo agrega")
	}
}
