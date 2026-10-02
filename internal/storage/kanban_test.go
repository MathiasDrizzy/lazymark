package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var std = Columns{"todo", "doing", "done"}

// TestColumnOf (K1): la columna de una tarea: sin tag es la primera, #kb/<col> la de ese id, [x] siempre es la de
// hecho (aunque lleve un tag), un tag desconocido cae en la primera, y el formato anterior (#doing, #wip,
// #progreso, #in-progress) se lee como #kb/doing.
func TestColumnOf(t *testing.T) {
	custom := Columns{"backlog", "doing", "review", "hecho"}
	cases := []struct {
		name string
		cols Columns
		line string
		want int
	}{
		{"sin tag", std, "- [ ] tarea", 0},
		{"#kb/doing", std, "- [ ] tarea #kb/doing", 1},
		{"#kb/todo explícito", std, "- [ ] tarea #kb/todo", 0},
		{"[x] sin tag", std, "- [x] tarea", 2},
		{"[X] mayúscula", std, "- [X] tarea", 2},
		{"[x] con tag de otra columna", std, "- [x] tarea #kb/doing", 2},
		{"tag desconocido cae en la primera", std, "- [ ] tarea #kb/revisar", 0},
		{"tag en mayúsculas", std, "- [ ] tarea #KB/Doing", 1},
		{"tag en medio del texto", std, "- [ ] tarea #kb/doing y más", 1},
		{"formato anterior #doing", std, "- [ ] tarea #doing", 1},
		{"formato anterior #wip", std, "- [ ] tarea #wip", 1},
		{"formato anterior #progreso", std, "- [ ] tarea #progreso", 1},
		{"formato anterior #in-progress", std, "- [ ] tarea #in-progress", 1},
		{"#kb manda sobre el formato anterior", std, "- [ ] tarea #doing #kb/todo", 0},
		{"un tag que solo empieza como doing", std, "- [ ] tarea #doingnow", 0},
		{"columnas configuradas: review", custom, "- [ ] tarea #kb/review", 2},
		{"columnas configuradas: la de hecho es 'hecho'", custom, "- [x] tarea", 3},
		{"columnas configuradas: la primera es backlog", custom, "- [ ] tarea", 0},
		{"formato anterior sin columna doing", Columns{"a", "b", "done"}, "- [ ] tarea #doing", 0},
		{"sin id done: la última es la de hecho", Columns{"a", "b", "c"}, "- [x] tarea", 2},
	}
	for _, c := range cases {
		tasks := (&Storage{}).extractTasks("n", "/n.md", c.line)
		if len(tasks) != 1 {
			t.Fatalf("%s: no se leyó la tarea de %q", c.name, c.line)
		}
		if got := c.cols.Of(tasks[0]); got != c.want {
			t.Errorf("%s: columna %d, se esperaba %d", c.name, got, c.want)
		}
	}
}

// TestRewriteForColumn (K1): mover cambia solo el tag y la casilla de esa línea.
func TestRewriteForColumn(t *testing.T) {
	custom := Columns{"backlog", "doing", "review", "done"}
	cases := []struct {
		name   string
		cols   Columns
		line   string
		target int
		want   string
	}{
		{"todo → doing agrega el tag al final", std, "- [ ] tarea", 1, "- [ ] tarea #kb/doing"},
		{"doing → todo quita el tag", std, "- [ ] tarea #kb/doing", 0, "- [ ] tarea"},
		{"doing → done marca [x] y quita el tag", std, "- [ ] tarea #kb/doing", 2, "- [x] tarea"},
		{"done → doing desmarca y pone el tag", std, "- [x] tarea", 1, "- [ ] tarea #kb/doing"},
		{"done con tag → todo", std, "- [x] tarea #kb/doing", 0, "- [ ] tarea"},
		{"done → done no cambia nada", std, "- [x] tarea", 2, "- [x] tarea"},
		{"mismo lugar no cambia nada", std, "- [ ] tarea #kb/doing", 1, "- [ ] tarea #kb/doing"},
		{"tag desconocido se reemplaza en su sitio", std, "- [ ] a #kb/revisar b", 1, "- [ ] a #kb/doing b"},
		{"tag desconocido y a done", std, "- [ ] a #kb/revisar", 2, "- [x] a"},
		{"formato anterior se migra al mover", std, "- [ ] tarea #doing", 0, "- [ ] tarea"},
		{"formato anterior a otra columna", custom, "- [ ] tarea #wip", 2, "- [ ] tarea #kb/review"},
		{"formato anterior a la misma columna", std, "- [ ] tarea #doing", 1, "- [ ] tarea #kb/doing"},
		{"columnas configuradas", custom, "- [ ] tarea", 2, "- [ ] tarea #kb/review"},
		{"la primera columna no lleva tag", custom, "- [ ] tarea #kb/review", 0, "- [ ] tarea"},
		{"bullet *, sangría y mayúscula", std, "  * [X] tarea", 1, "  * [ ] tarea #kb/doing"},
		{"CRLF se conserva", std, "- [ ] tarea\r", 1, "- [ ] tarea #kb/doing\r"},
		{"espacios al final se conservan", std, "- [ ] tarea  ", 1, "- [ ] tarea #kb/doing  "},
		{"texto con emoji y fechas", std, "- [ ] enviar 📅 2026-10-05", 1, "- [ ] enviar 📅 2026-10-05 #kb/doing"},
		{"otros tags no se tocan", std, "- [ ] tarea #diseño #kb/doing #urgente", 2, "- [x] tarea #diseño #urgente"},
	}
	for _, c := range cases {
		got, err := RewriteForColumn(c.line, c.cols, c.target)
		if err != nil || got != c.want {
			t.Errorf("%s:\n  %q →(col %d) %q (err %v)\n  se esperaba %q", c.name, c.line, c.target, got, err, c.want)
		}
	}
	for _, bad := range []string{"texto", "- tarea", "- [-] tarea", "# - [ ] x", ""} {
		if _, err := RewriteForColumn(bad, std, 1); err == nil {
			t.Errorf("%q no es una tarea: debía dar error", bad)
		}
	}
	if _, err := RewriteForColumn("- [ ] t", std, 3); err == nil {
		t.Error("una columna fuera de rango debe dar error")
	}
}

func kanbanNote(t *testing.T, body string) (*Storage, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "n.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return New(dir), p
}

// TestMoveTaskChangesOneLine (K1): mover una tarjeta cambia exactamente esa línea del archivo; el resto queda
// idéntico byte a byte (diff de 1 línea).
func TestMoveTaskChangesOneLine(t *testing.T) {
	body := "# Plan\n\nTexto con #diseño.\n- [ ] uno\n- [ ] dos #kb/doing\n  - [ ] sub\n\n- [x] tres\nfin sin salto"
	s, p := kanbanNote(t, body)
	if err := s.MoveTask(p, 4, std, 1, time.Time{}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	bl, al := strings.Split(body, "\n"), strings.Split(string(after), "\n")
	if len(bl) != len(al) {
		t.Fatalf("cambió el número de líneas: %d → %d", len(bl), len(al))
	}
	changed := 0
	for i := range bl {
		if bl[i] != al[i] {
			changed++
			if i != 3 || al[i] != "- [ ] uno #kb/doing" {
				t.Errorf("cambió la línea %d: %q → %q", i+1, bl[i], al[i])
			}
		}
	}
	if changed != 1 {
		t.Errorf("deben cambiar solo 1 línea, cambiaron %d", changed)
	}
	// a done y de vuelta; sigue siendo 1 línea
	if err := s.MoveTask(p, 4, std, 2, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "\n- [x] uno\n") {
		t.Errorf("no quedó en done:\n%s", b)
	}
}

// TestMoveTaskDetectsChangedNote (K2): si la nota cambió en disco entre la lectura y la escritura, no se pisa:
// devuelve ErrNoteChanged y el archivo queda como lo dejó el otro programa.
func TestMoveTaskDetectsChangedNote(t *testing.T) {
	s, p := kanbanNote(t, "- [ ] uno\n- [ ] dos\n")
	st, _ := os.Stat(p)
	read := st.ModTime()
	// otro programa edita la nota después de que lazymark la leyó
	external := "- [ ] uno\n- [ ] dos editada afuera\n"
	os.WriteFile(p, []byte(external), 0o644)
	later := read.Add(2 * time.Second)
	os.Chtimes(p, later, later)
	if err := s.MoveTask(p, 2, std, 1, read); !errors.Is(err, ErrNoteChanged) {
		t.Fatalf("debía dar ErrNoteChanged: %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != external {
		t.Errorf("se pisó el cambio externo: %q", b)
	}
	// con el mtime al día sí escribe
	st, _ = os.Stat(p)
	if err := s.MoveTask(p, 2, std, 1, st.ModTime()); err != nil {
		t.Errorf("con el mtime actual debía escribir: %v", err)
	}
}

// TestKanbanTagIsNotACategory: #kb/<col> no sale en las categorías (si no, habría una categoría "kb").
func TestKanbanTagIsNotACategory(t *testing.T) {
	s := New(t.TempDir())
	tags := s.extractTags("- [ ] a #kb/doing\n- [ ] b #kb/todo #diseno\n")
	if len(tags) != 1 || tags[0] != "diseno" {
		t.Errorf("categorías = %v, solo debía salir diseno", tags)
	}
}

func TestCleanTaskText(t *testing.T) {
	for in, want := range map[string]string{
		"tarea #kb/doing":           "tarea",
		"tarea #KB/review y más":    "tarea y más",
		"tarea #doing":              "tarea",
		"tarea #wip #urgente":       "tarea #urgente",
		"a #kb/x #kb/y":             "a",
		"nada que limpiar":          "nada que limpiar",
		"#kb/doing al principio":    "al principio",
		"un #doingnow que se queda": "un #doingnow que se queda",
	} {
		if got := CleanTaskText(in); got != want {
			t.Errorf("CleanTaskText(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

func TestMoveTaskRefusesOutsideNotes(t *testing.T) {
	s, _ := kanbanNote(t, "- [ ] a\n")
	outside := filepath.Join(t.TempDir(), "fuera.md")
	os.WriteFile(outside, []byte("- [ ] x\n"), 0o644)
	if err := s.MoveTask(outside, 1, std, 1, time.Time{}); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("fuera de la carpeta: %v", err)
	}
}

// TestTaskIDsAreStable (H4-3): el id de una tarea no cambia al editar o insertar otras líneas, al moverla de columna ni
// al marcarla; las repetidas se distinguen con .n; y cambia si se edita su texto.
func TestTaskIDsAreStable(t *testing.T) {
	s, p := kanbanNote(t, "# T\n- [ ] comprar leche\n- [ ] llamar\n- [ ] comprar leche\n")
	idsOf := func() map[string]Task {
		notes, _ := s.ListNotes()
		out := map[string]Task{}
		for i, id := range s.TaskIDs(notes[0]) {
			out[id] = notes[0].Tasks[i]
		}
		return out
	}
	first := idsOf()
	if len(first) != 3 {
		t.Fatalf("3 tareas con 3 ids distintos: %v", first)
	}
	var leche1, llamar, leche2 string
	for id, tk := range first {
		switch {
		case tk.Text == "llamar":
			llamar = id
		case strings.HasSuffix(id, ".2"):
			leche2 = id
		default:
			leche1 = id
		}
	}
	if !strings.HasPrefix(llamar, "n.md#") || leche1+".2" != leche2 {
		t.Fatalf("ids inesperados: %v", first)
	}
	// insertar y editar otras líneas, y mover/marcar la tarea
	os.WriteFile(p, []byte("# Otro título\nnueva línea\n\n- [ ] comprar leche\n- [ ] llamar\n- [ ] comprar leche\ntexto\n"), 0o644)
	if _, ok := idsOf()[llamar]; !ok {
		t.Error("el id debe sobrevivir a insertar y editar otras líneas")
	}
	notes, _ := s.ListNotes()
	var line int
	for _, tk := range notes[0].Tasks {
		if tk.Text == "llamar" {
			line = tk.Line
		}
	}
	if err := s.MoveTask(p, line, std, 1, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveTask(p, line, std, 2, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, ok := idsOf()[llamar]; !ok {
		t.Error("el id debe sobrevivir a mover de columna y marcar la tarea")
	}
	// editar su propio texto sí lo cambia
	b, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(b), "llamar", "llamar al banco", 1)), 0o644)
	if _, ok := idsOf()[llamar]; ok {
		t.Error("editar el texto de la tarea cambia su id")
	}
	// FindTask
	n, tk, err := s.FindTask(leche2)
	if err != nil || tk.Line != 6 || n.Title == "" {
		t.Errorf("FindTask(%s) = %v %v %v", leche2, tk, n.Title, err)
	}
	for _, bad := range []string{"", "n.md", "n.md#", "#abc", "n.md#00000000", "nada.md#" + strings.Split(leche1, "#")[1]} {
		if _, _, err := s.FindTask(bad); !errors.Is(err, ErrTaskNotFound) {
			t.Errorf("FindTask(%q) debía dar ErrTaskNotFound: %v", bad, err)
		}
	}
}

func TestResolveFolder(t *testing.T) {
	notes, _, outside := layout(t)
	s := New(notes)
	if got, err := s.ResolveFolder(""); err != nil || got == "" {
		t.Errorf("la carpeta de notas misma: %v", err)
	}
	if _, err := s.ResolveFolder("sub"); err != nil {
		t.Errorf("sub: %v", err)
	}
	for _, bad := range []string{"..", "../x", "sub/../..", filepath.Dir(outside), "nada", "sub/a.md"} {
		if _, err := s.ResolveFolder(bad); err == nil {
			t.Errorf("%q debía rechazarse", bad)
		}
	}
}
