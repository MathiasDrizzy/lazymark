package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/ops"
)

var update = flag.Bool("update", false, "reescribe los golden JSON de testdata/golden")

// fixture copia testdata/notas a una carpeta temporal y aísla la config del usuario (HOME y XDG): los tests
// nunca tocan las notas ni la configuración reales.
func fixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	dst := filepath.Join(t.TempDir(), "notas")
	err := filepath.WalkDir("testdata/notas", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata/notas", p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	dst, _ = filepath.EvalSymlinks(dst)
	return dst
}

// snapshot es el contenido de todos los archivos y carpetas de dir, para comprobar que un comando no tocó nada.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			out[rel+"/"] = ""
			return nil
		}
		b, _ := os.ReadFile(p)
		out[rel] = string(b)
		return nil
	})
	return out
}

var timeRe = regexp.MustCompile(`"mod_time": "[^"]*"`)

// normalize vuelve estable la salida: la carpeta temporal es <DIR> y las horas <TIME>.
func normalize(out, dir string) string {
	// en JSON las barras de Windows van escapadas (\\): se normaliza la carpeta y el resto de la ruta a "/"
	out = strings.ReplaceAll(out, strings.ReplaceAll(dir, `\`, `\\`), "<DIR>")
	out = strings.ReplaceAll(out, dir, "<DIR>")
	out = dirTailRe.ReplaceAllStringFunc(out, func(m string) string { return strings.ReplaceAll(m, `\\`, "/") })
	return timeRe.ReplaceAllString(out, `"mod_time": "<TIME>"`)
}

var dirTailRe = regexp.MustCompile(`<DIR>[^"]*`)

func run(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	var err error
	args = append(args, "--dir", dir)
	switch args[0] {
	case "note":
		err = RunNoteWithWriter(&buf, args[1:], dir)
	case "task":
		err = RunTaskWithWriter(&buf, args[1:], dir)
	default:
		t.Fatalf("comando %q", args[0])
	}
	return buf.String(), err
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", "golden", name)
	if *update {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("falta el golden %s (go test ./internal/cli -update): %v", p, err)
	}
	if string(want) != got {
		t.Errorf("la salida de %s cambió (el esquema JSON es estable: docs/cli.md):\n--- golden\n%s\n--- actual\n%s", name, want, got)
	}
}

// taskID busca el id de la tarea con ese texto en la salida de `task list --json`.
func taskID(t *testing.T, dir, text, note string) string {
	t.Helper()
	out, err := run(t, dir, "task", "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tasks []ops.TaskDTO
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatal(err)
	}
	for _, tk := range tasks {
		if tk.Text == text && tk.Note == note {
			return tk.ID
		}
	}
	t.Fatalf("no hay la tarea %q de %s en %s", text, note, out)
	return ""
}

// TestGoldenJSON (CL1): el JSON de cada comando es estable, campo por campo.
func TestGoldenJSON(t *testing.T) {
	dir := fixture(t)
	for _, c := range []struct {
		golden string
		args   []string
	}{
		{"note-list.json", []string{"note", "list", "--json"}},
		{"note-show.json", []string{"note", "show", "ideas.md", "--json"}},
		{"task-list.json", []string{"task", "list", "--json"}},
		{"task-list-pending-doing.json", []string{"task", "list", "--json", "--pending", "--column", "doing"}},
		{"task-list-note.json", []string{"task", "list", "--json", "--note", "ideas.md"}},
	} {
		out, err := run(t, dir, c.args...)
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		golden(t, c.golden, normalize(out, dir))
	}

	// mover, marcar y crear cambian el disco: se verifican en este orden
	id := taskID(t, dir, "Escribir informe", "proyecto.md")
	out, err := run(t, dir, "task", "move", id, "doing", "--json")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "task-move.json", normalize(out, dir))
	out, err = run(t, dir, "task", "toggle", id, "--json")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "task-toggle.json", normalize(out, dir))
	out, err = run(t, dir, "note", "new", "Nota nueva", "--folder", "sub", "--json")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "note-new.json", normalize(out, dir))
}

// TestPlainOutput: sin --json, una línea legible por elemento.
func TestPlainOutput(t *testing.T) {
	dir := fixture(t)
	out, err := run(t, dir, "task", "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[ ] proyecto.md#", "Escribir informe", "(doing)", "(todo)", "[x] proyecto.md#", "Publicar versión", "(done)"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida de texto no contiene %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{") || strings.Contains(out, "#kb/") {
		t.Errorf("la salida de texto no es JSON ni muestra las etiquetas del tablero:\n%s", out)
	}
	out, _ = run(t, dir, "note", "list")
	if !strings.Contains(out, "proyecto ("+filepath.Join(dir, "proyecto.md")+")") {
		t.Errorf("note list: %s", out)
	}
	out, _ = run(t, dir, "note", "show", "ideas.md")
	if !strings.HasPrefix(out, "# Ideas") {
		t.Errorf("note show imprime el contenido: %q", out)
	}
	id := taskID(t, dir, "Comprar café", "ideas.md")
	out, err = run(t, dir, "task", "move", id, "done")
	if err != nil || out != id+" → done\n" {
		t.Errorf("task move: %q %v", out, err)
	}
}

// TestMoveWritesOnlyThatLine: task move reescribe solo la línea de la tarea, con el tag de la columna.
func TestMoveWritesOnlyThatLine(t *testing.T) {
	dir := fixture(t)
	before, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	id := taskID(t, dir, "Revisar código", "proyecto.md")
	if _, err := run(t, dir, "task", "move", id, "todo"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	want := strings.Replace(string(before), "- [ ] Revisar código #kb/doing", "- [ ] Revisar código", 1)
	if string(after) != want {
		t.Errorf("proyecto.md:\n%s\nse esperaba:\n%s", after, want)
	}
	// el id no cambió al moverla, y se acepta el título visible de la columna
	if _, err := run(t, dir, "task", "move", id, "En progreso"); err != nil {
		t.Errorf("mover por título visible: %v", err)
	}
	// la tarea heredada #wip se migra a #kb/doing al mover
	legacy := taskID(t, dir, "Tarea heredada", "proyecto.md")
	if _, err := run(t, dir, "task", "move", legacy, "done"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	if !strings.Contains(string(b), "- [x] Tarea heredada\n") {
		t.Errorf("la etiqueta heredada debía desaparecer al pasar a hecho:\n%s", b)
	}
}

// TestNewNote: crea la nota en la carpeta pedida, con plantilla o vacía.
func TestNewNote(t *testing.T) {
	dir := fixture(t)
	out, err := run(t, dir, "note", "new", "Mi", "nota", "--folder", "sub")
	if err != nil || strings.TrimSpace(out) != filepath.Join(dir, "sub", "mi-nota.md") {
		t.Fatalf("note new: %q %v", out, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "sub", "mi-nota.md")); !strings.Contains(string(b), "- [ ]") {
		t.Errorf("con plantilla: %q", b)
	}
	out, err = run(t, dir, "note", "new", "Vacia", "--empty")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(strings.TrimSpace(out)); string(b) != "# Vacia\n" {
		t.Errorf("--empty: %q", b)
	}
}

// TestInvalidArgsExit2NoSideEffects (CL1): los argumentos inválidos salen con 2, sin stdout y sin tocar nada del
// disco (ni siquiera crear la carpeta de notas).
func TestInvalidArgsExit2NoSideEffects(t *testing.T) {
	dir := fixture(t)
	before := snapshot(t, dir)
	id := taskID(t, dir, "Escribir informe", "proyecto.md")
	cases := map[string][]string{
		"nota sin subcomando":      {"note"},
		"tarea sin subcomando":     {"task"},
		"subcomando desconocido":   {"task", "borrar"},
		"nota desconocida":         {"note", "borrar"},
		"flag desconocido":         {"task", "list", "--nope"},
		"flag sin valor":           {"task", "list", "--column"},
		"move sin columna":         {"task", "move", id},
		"move sin nada":            {"task", "move"},
		"move columna inexistente": {"task", "move", id, "cancelada"},
		"move de más":              {"task", "move", id, "doing", "extra"},
		"toggle sin id":            {"task", "toggle"},
		"toggle path sin line":     {"task", "toggle", "--path", "proyecto.md"},
		"list con posicional":      {"task", "list", "algo"},
		"list columna inexistente": {"task", "list", "--column", "nope"},
		"show sin ruta":            {"note", "show"},
		"show fuera":               {"note", "show", "../x.md"},
		"show no .md":              {"note", "show", "/etc/hosts"},
		"new sin título":           {"note", "new"},
		"new título en blanco":     {"note", "new", "   "},
		"new duplicada":            {"note", "new", "Ideas"},
		"new carpeta fuera":        {"note", "new", "X", "--folder", ".."},
		"new carpeta absoluta":     {"note", "new", "X", "--folder", "/tmp"},
		"new carpeta inexistente":  {"note", "new", "X", "--folder", "nada"},
		"list con posicional nota": {"note", "list", "x"},
	}
	for name, args := range cases {
		out, err := run(t, dir, args...)
		if ExitCode(err) != ExitUsage {
			t.Errorf("%s: código %d (se esperaba 2), err=%v", name, ExitCode(err), err)
		}
		if out != "" {
			t.Errorf("%s: stdout debía estar vacío: %q", name, out)
		}
	}
	after := snapshot(t, dir)
	if len(after) != len(before) {
		t.Errorf("los comandos inválidos cambiaron la carpeta: %d → %d entradas", len(before), len(after))
	}
	for k, v := range before {
		if after[k] != v {
			t.Errorf("%s cambió", k)
		}
	}
	// una carpeta de notas inexistente no se crea
	ghost := filepath.Join(t.TempDir(), "no-existe")
	for _, args := range [][]string{{"task", "move"}, {"note", "show"}, {"task", "list", "--nope"}} {
		run(t, ghost, args...)
	}
	if _, err := os.Stat(ghost); err == nil {
		t.Error("un comando inválido creó la carpeta de notas")
	}
}

// TestNotFoundExit3: una tarea o una nota que no existen salen con 3.
func TestNotFoundExit3(t *testing.T) {
	dir := fixture(t)
	for name, args := range map[string][]string{
		"tarea":           {"task", "toggle", "proyecto.md#00000000"},
		"tarea sin #":     {"task", "move", "inventado", "doing"},
		"nota":            {"note", "show", "nada.md"},
		"línea sin tarea": {"task", "toggle", "--path", "proyecto.md", "--line", "1"},
	} {
		if _, err := run(t, dir, args...); ExitCode(err) != ExitNotFound {
			t.Errorf("%s: código %d (se esperaba 3): %v", name, ExitCode(err), err)
		}
	}
}

// TestHelp: -h imprime el uso y los códigos de salida y no falla.
func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"task", "-h"}, {"task", "move", "--help"}, {"note", "--help"}, {"note", "list", "-h"}} {
		var buf bytes.Buffer
		var err error
		if args[0] == "task" {
			err = RunTaskWithWriter(&buf, args[1:], t.TempDir())
		} else {
			err = RunNoteWithWriter(&buf, args[1:], t.TempDir())
		}
		if err != nil || !strings.Contains(buf.String(), "lazymark "+args[0]) || !strings.Contains(buf.String(), "3") {
			t.Errorf("%v: %v %q", args, err, buf.String())
		}
	}
}

// TestLegacyToggleByLine: la forma anterior `task toggle --path <nota> --line <n>` sigue funcionando.
func TestLegacyToggleByLine(t *testing.T) {
	dir := fixture(t)
	if _, err := run(t, dir, "task", "toggle", "--path", filepath.Join(dir, "proyecto.md"), "--line", "5"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "proyecto.md")); !strings.Contains(string(b), "- [x] Escribir informe") {
		t.Errorf("no se marcó:\n%s", b)
	}
}

// TestTextOutputStripsControlChars (S5): la salida de texto no lleva escapes de terminal del contenido de las notas
// (OSC 52 escribiría en el portapapeles); --json los escapa y conserva el texto exacto.
func TestTextOutputStripsControlChars(t *testing.T) {
	dir := fixture(t)
	evil := "# e\x1b]0;titulo\x07\n- [ ] x \x1b]52;c;cHduZWQ=\x07 y\x9b31m z\r\n"
	if err := os.WriteFile(filepath.Join(dir, "e.md"), []byte(evil), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"task", "list"}, {"note", "list"}, {"note", "show", "e.md"}} {
		out, err := run(t, dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range out {
			if r != '\n' && r != '\t' && (r < 0x20 || (r >= 0x7f && r <= 0x9f)) {
				t.Errorf("%v: la salida lleva el carácter de control %q:\n%q", args, r, out)
				break
			}
		}
	}
	out, _ := run(t, dir, "task", "list")
	if !strings.Contains(out, "x ]52;c;cHduZWQ= y") {
		t.Errorf("el texto visible se conserva: %q", out)
	}
	out, _ = run(t, dir, "task", "list", "--json")
	if !strings.Contains(out, `\u001b]52`) {
		t.Errorf("--json conserva el texto exacto, escapado: %q", out)
	}
}

// TestNewNoteRejectsControlCharsInTitle (S7): un título con salto de línea no escribe contenido arbitrario.
func TestNewNoteRejectsControlCharsInTitle(t *testing.T) {
	dir := fixture(t)
	before := snapshot(t, dir)
	for _, title := range []string{"t\n- [ ] inyectada", "t\r\nx", "t\x1b[0m", strings.Repeat("a", 201)} {
		if _, err := run(t, dir, "note", "new", title, "--empty"); ExitCode(err) != ExitUsage {
			t.Errorf("%q: código %d: %v", title, ExitCode(err), err)
		}
	}
	if after := snapshot(t, dir); len(after) != len(before) {
		t.Error("no debía crearse nada")
	}
}
