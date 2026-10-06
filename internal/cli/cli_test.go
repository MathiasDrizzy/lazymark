package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/ops"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

func init() { storage.Today = func() string { return "2026-10-02" } } // las pruebas no dependen del reloj

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
	setDateFormat(t, dst, "emoji") // estas pruebas ejercitan el formato de emojis; las de Dataview lo cambian
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
	case "search":
		err = RunSearchWithWriter(&buf, args[1:], dir)
	case "note":
		err = RunNoteWithWriter(&buf, args[1:], dir)
	case "task":
		err = RunTaskWithWriter(&buf, args[1:], dir)
	case "daily":
		err = RunDailyWithWriter(&buf, args[1:], dir)
	case "dates":
		err = RunDatesWithWriter(&buf, args[1:], dir)
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
	if err != nil || out != id+" → done  ✓ 2026-10-02\n" {
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
	if !strings.Contains(string(b), "- [x] Tarea heredada ✅ 2026-10-02\n") {
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

// TestTaskDueAndStart (C.6): `task due` y `task start` ponen y quitan las fechas (formato de Obsidian Tasks) cambiando solo
// la línea de la tarea; el JSON lleva start, due, completed y overdue; una fecha inválida sale con 2 sin tocar nada.
func TestTaskDueAndStart(t *testing.T) {
	dir := fixture(t)
	id := taskID(t, dir, "Escribir informe", "proyecto.md")
	before, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))

	out, err := run(t, dir, "task", "due", id, "2020-01-02", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var tk ops.TaskDTO
	if err := json.Unmarshal([]byte(out), &tk); err != nil {
		t.Fatal(err)
	}
	if tk.Due != "2020-01-02" || !tk.Overdue || tk.Start != "" || tk.Completed != "" || tk.ID != id {
		t.Errorf("tras due: %+v", tk)
	}
	if _, err := run(t, dir, "task", "start", id, "2019-12-01"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	want := strings.Replace(string(before), "- [ ] Escribir informe\n", "- [ ] Escribir informe 📅 2020-01-02 🛫 2019-12-01\n", 1)
	if string(after) != want {
		t.Errorf("proyecto.md:\n%s\nse esperaba:\n%s", after, want)
	}
	out, _ = run(t, dir, "task", "list", "--note", "proyecto.md")
	if !strings.Contains(out, "▸ 2019-12-01 ◷ 2020-01-02 (") || strings.Contains(out, "◷ 2020-01-02 ▸") || strings.ContainsAny(out, "🛫📅✅") {
		t.Errorf("la salida de texto muestra las fechas: %s", out)
	}
	// quitar con none
	if _, err := run(t, dir, "task", "due", id, "none"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, dir, "task", "start", id, "NONE"); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(filepath.Join(dir, "proyecto.md")); string(again) != string(before) {
		t.Errorf("tras quitar las dos fechas la nota debía quedar idéntica:\n%s", again)
	}
	// marcar como hecha agrega ✅ y moverla otra vez a todo lo quita
	if _, err := run(t, dir, "task", "move", id, "done"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	if !strings.Contains(string(b), "- [x] Escribir informe ✅ 2026-10-02") {
		t.Errorf("al pasar a hecho se agrega ✅:\n%s", b)
	}
	if _, err := run(t, dir, "task", "move", id, "todo"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "proyecto.md")); string(b) != string(before) {
		t.Errorf("al volver a todo se quita ✅:\n%s", b)
	}
	// argumentos inválidos: exit 2 y la carpeta intacta
	snap := snapshot(t, dir)
	for name, args := range map[string][]string{
		"fecha inexistente": {"task", "due", id, "2026-02-30"},
		"formato":           {"task", "start", id, "mañana"},
		"sin fecha":         {"task", "due", id},
		"sin nada":          {"task", "due"},
		"de más":            {"task", "start", id, "2026-01-01", "x"},
	} {
		if _, err := run(t, dir, args...); ExitCode(err) != ExitUsage {
			t.Errorf("%s: código %d: %v", name, ExitCode(err), err)
		}
	}
	if _, err := run(t, dir, "task", "due", "proyecto.md#00000000", "2026-01-01"); ExitCode(err) != ExitNotFound {
		t.Errorf("tarea inexistente: código %d", ExitCode(err))
	}
	for k, v := range snap {
		if now := snapshot(t, dir)[k]; now != v {
			t.Errorf("%s cambió", k)
		}
	}
}

// TestTaskDateDefectsOfTheAudit (C.7): los tres casos exactos de la auditoría de C.5–C.6, con la CLI.
//   - F1: una fecha inválida con el mismo emoji se reemplaza: nunca quedan dos marcadores del mismo campo.
//   - F2: `none` quita todos los marcadores del campo y el código de salida refleja lo que pasó (0 si escribió, y el id nuevo
//     va en la salida y en --json).
//   - F3: al quitar una fecha el texto no queda pegado a otro emoji.
func TestTaskDateDefectsOfTheAudit(t *testing.T) {
	dir := fixture(t)
	notePath := filepath.Join(dir, "fechas.md")
	write := func(body string) { os.WriteFile(notePath, []byte(body), 0o644) }
	read := func() string { b, _ := os.ReadFile(notePath); return string(b) }
	idOf := func(text string) string {
		out, err := run(t, dir, "task", "list", "--json", "--note", "fechas.md")
		if err != nil {
			t.Fatal(err)
		}
		var tasks []ops.TaskDTO
		json.Unmarshal([]byte(out), &tasks)
		for _, tk := range tasks {
			if strings.HasPrefix(tk.Text, text) {
				return tk.ID
			}
		}
		t.Fatalf("no hay la tarea %q en %s", text, out)
		return ""
	}

	// F1
	write("# F\n- [ ] invalida 📅 2026-13-45\n")
	if _, err := run(t, dir, "task", "due", idOf("invalida"), "2026-10-10"); err != nil {
		t.Fatalf("F1: %v", err)
	}
	if got := read(); got != "# F\n- [ ] invalida 📅 2026-10-10\n" {
		t.Errorf("F1: la fecha inválida debía reemplazarse sin duplicar el emoji:\n%q", got)
	}

	// F2: dos 📅 y `none`: quita las dos, sale con 0 y el id de la salida es el de la tarea ahora
	write("# F\n- [ ] doble 📅 2026-10-01 📅 2026-10-02\n")
	before := idOf("doble")
	out, err := run(t, dir, "task", "due", before, "none", "--json")
	if err != nil || ExitCode(err) != 0 {
		t.Fatalf("F2: salió con %d aunque escribió: %v", ExitCode(err), err)
	}
	if got := read(); got != "# F\n- [ ] doble\n" {
		t.Errorf("F2: none debía quitar todos los 📅:\n%q", got)
	}
	var tk ops.TaskDTO
	if err := json.Unmarshal([]byte(out), &tk); err != nil {
		t.Fatal(err)
	}
	if now := idOf("doble"); tk.ID != now || tk.Due != "" {
		t.Errorf("F2: el JSON debe llevar el id nuevo %q y no tener fecha: %+v", now, tk)
	}
	write("# F\n- [ ] doble 📅 2026-10-01 📅 2026-10-02\n")
	text, err := run(t, dir, "task", "due", idOf("doble"), "none")
	if err != nil || !strings.HasPrefix(text, idOf("doble")+" → ") {
		t.Errorf("F2: la salida de texto lleva el id nuevo: %q %v", text, err)
	}

	// F3
	write("# F\n- [ ] pegadas 🛫 2026-10-01📅 2026-10-03\n")
	if _, err := run(t, dir, "task", "start", idOf("pegadas"), "none"); err != nil {
		t.Fatalf("F3: %v", err)
	}
	if got := read(); got != "# F\n- [ ] pegadas 📅 2026-10-03\n" {
		t.Errorf("F3: el texto no debe quedar pegado al emoji:\n%q", got)
	}
}

// TestDateRemovalKeepsAChecklistItem (C.7): quitar la única fecha de una tarea que no tiene más texto no la hace desaparecer
// (la línea sigue siendo "- [ ] "): el comando sale con 0 y devuelve la tarea.
func TestDateRemovalKeepsAChecklistItem(t *testing.T) {
	dir := fixture(t)
	p := filepath.Join(dir, "solo.md")
	os.WriteFile(p, []byte("# S\n- [ ] 📅 2026-05-10\n"), 0o644)
	out, err := run(t, dir, "task", "list", "--json", "--note", "solo.md")
	if err != nil {
		t.Fatal(err)
	}
	var tasks []ops.TaskDTO
	json.Unmarshal([]byte(out), &tasks)
	if len(tasks) != 1 {
		t.Fatalf("la tarea de solo fecha debe listarse: %s", out)
	}
	res, err := run(t, dir, "task", "due", tasks[0].ID, "none", "--json")
	if err != nil {
		t.Fatalf("salió con %d: %v", ExitCode(err), err)
	}
	var got ops.TaskDTO
	if json.Unmarshal([]byte(res), &got) != nil || got.Due != "" || got.Text != "" || got.ID != tasks[0].ID {
		t.Errorf("debe devolver la tarea como quedó (sin fecha ni texto): %s", res)
	}
	if b, _ := os.ReadFile(p); string(b) != "# S\n- [ ] \n" {
		t.Errorf("la nota quedó %q", b)
	}
}

// TestSearchCLI (C.1 H5-1): `lazymark search` con y sin --json, --regex, --case y --limit; sin coincidencias no es un error;
// una búsqueda vacía o una expresión inválida salen con 2 sin tocar nada; la salida de texto no lleva caracteres de control.
func TestSearchCLI(t *testing.T) {
	dir := fixture(t)
	os.WriteFile(filepath.Join(dir, "ctrl.md"), []byte("# c\nescribir \x1b]52;c;cHduZWQ=\x07 informe\n"), 0o644)
	out, err := run(t, dir, "search", "informe")
	if err != nil {
		t.Fatal(err)
	}
	want := "ctrl.md:2: escribir ]52;c;cHduZWQ= informe\nideas.md:3: - [ ] Escribir informe\nproyecto.md:5: - [ ] Escribir informe\n"
	if out != want {
		t.Errorf("texto:\n%q\nse esperaba:\n%q", out, want)
	}
	for _, args := range [][]string{{"search", "informe", "--json"}, {"search", "-regex", `Escribir \w+`, "--json"}} {
		o, err := run(t, dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		var res ops.SearchDTO
		if err := json.Unmarshal([]byte(o), &res); err != nil || len(res.Matches) < 2 {
			t.Fatalf("%v: %v %s", args, err, o)
		}
		for _, m := range res.Matches {
			if m.Text[m.Start:m.End] == "" || !strings.EqualFold(m.Text[m.Start:m.End], "informe") && args[1] == "informe" {
				t.Errorf("%v: el rango %d:%d de %q no es lo hallado", args, m.Start, m.End, m.Text)
			}
		}
		if args[len(args)-1] == "--json" && args[1] == "informe" {
			golden(t, "search.json", normalize(o, dir))
		}
	}
	if o, err := run(t, dir, "search", "ESCRIBIR", "--case"); err != nil || o != "" {
		t.Errorf("--case no halla ESCRIBIR: %q %v", o, err)
	}
	if o, err := run(t, dir, "search", "nada de nada"); err != nil || o != "" {
		t.Errorf("sin coincidencias sale con 0 y sin salida: %q %v", o, err)
	}
	if o, _ := run(t, dir, "search", "informe", "--limit", "1", "--json"); !strings.Contains(o, `"truncated": true`) {
		t.Errorf("--limit 1 debía truncar: %s", o)
	}
	// varias palabras sin comillas son la misma búsqueda
	if o, _ := run(t, dir, "search", "Escribir", "informe"); !strings.Contains(o, "ideas.md:3") {
		t.Errorf("varias palabras: %q", o)
	}
	snap := snapshot(t, dir)
	for name, args := range map[string][]string{
		"vacía":              {"search"},
		"regex inválida":     {"search", "(", "--regex"},
		"límite negativo":    {"search", "x", "--limit", "-3"},
		"flag desconocido":   {"search", "x", "--nope"},
		"límite no numérico": {"search", "x", "--limit", "mucho"},
	} {
		o, err := run(t, dir, args...)
		if ExitCode(err) != ExitUsage || o != "" {
			t.Errorf("%s: código %d, salida %q: %v", name, ExitCode(err), o, err)
		}
	}
	if o, err := run(t, dir, "search", "-h"); err != nil || !strings.Contains(o, "lazymark search") {
		t.Errorf("-h: %q %v", o, err)
	}
	for k, v := range snap {
		if snapshot(t, dir)[k] != v {
			t.Errorf("%s cambió", k)
		}
	}
}

// TestNoteNewFromTemplate (C.5): `note new --template` rellena {{date}}, {{time}} y {{title}}; una plantilla que no existe sale con 3 sin crear nada;
// un nombre con ruta no vale (2/3) y no lee fuera de templates/.
func TestNoteNewFromTemplate(t *testing.T) {
	dir := fixture(t)
	os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	os.WriteFile(filepath.Join(dir, "templates", "reunion.md"), []byte("# {{title}}\n\n{{date}} {{time}}\n"), 0o644)
	os.WriteFile(filepath.Join(filepath.Dir(dir), "secreto.md"), []byte("fuera"), 0o644)
	out, err := run(t, dir, "note", "new", "Equipo", "--template", "reunion")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(strings.TrimSpace(out))
	if ok, _ := regexp.MatchString(`^# Equipo\n\n\d{4}-\d\d-\d\d \d\d:\d\d\n$`, string(b)); !ok {
		t.Errorf("contenido: %q", b)
	}
	before := snapshot(t, dir)
	for _, bad := range []string{"nada", "../secreto", "../../secreto"} {
		if _, err := run(t, dir, "note", "new", "Otra", "--template", bad); err == nil || ExitCode(err) != 3 {
			t.Errorf("--template %q: %v (código %d), se esperaba 3", bad, err, ExitCode(err))
		}
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("una plantilla inválida no debe crear nada")
	}
}

// TestDailyCLI (C.5): `lazymark daily` crea journal/AAAA-MM-DD.md con la plantilla y la segunda vez la abre sin tocarla; con --json dice si la creó.
func TestDailyCLI(t *testing.T) {
	dir := fixture(t)
	os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	os.WriteFile(filepath.Join(dir, "templates", "daily.md"), []byte("# Diario {{date}}\n\n- [ ] revisar\n"), 0o644)
	today := time.Now().Format("2006-01-02")
	want := filepath.Join(dir, "journal", today+".md")
	out, err := run(t, dir, "daily")
	if err != nil || strings.TrimSpace(out) != want {
		t.Fatalf("daily: %q %v", out, err)
	}
	if b, _ := os.ReadFile(want); string(b) != "# Diario "+today+"\n\n- [ ] revisar\n" {
		t.Errorf("contenido: %q", b)
	}
	os.WriteFile(want, []byte("# editada\n"), 0o644)
	out, err = run(t, dir, "daily", "--json")
	var d struct {
		Path    string `json:"path"`
		Created bool   `json:"created"`
	}
	if err != nil || json.Unmarshal([]byte(out), &d) != nil || d.Path != want || d.Created {
		t.Fatalf("daily --json: %q %v %+v", out, err, d)
	}
	if b, _ := os.ReadFile(want); string(b) != "# editada\n" {
		t.Errorf("la segunda vez no debe tocarla: %q", b)
	}
	if _, err := run(t, dir, "daily", "sobra"); err == nil || ExitCode(err) != 2 {
		t.Errorf("un argumento de más es de uso: %v", err)
	}
}

// TestTemplateErrorsAndWarningsCLI (ORD-014 M1 y M2): una plantilla binaria, UTF-16 o de más de 256 KB sale con 2 y no crea nada (note new y
// daily); una variable desconocida deja la nota con la variable tal cual, avisa por stderr y en el JSON, y no cambia el código de salida.
func TestTemplateErrorsAndWarningsCLI(t *testing.T) {
	dir := fixture(t)
	tpl := filepath.Join(dir, "templates")
	os.MkdirAll(tpl, 0o755)
	os.WriteFile(filepath.Join(tpl, "bin.md"), []byte("# x\x00\x01\x02"), 0o644)
	os.WriteFile(filepath.Join(tpl, "utf16.md"), []byte{0xff, 0xfe, '#', 0, ' ', 0, 'h', 0}, 0o644)
	os.WriteFile(filepath.Join(tpl, "grande.md"), []byte("# x\n"+strings.Repeat("0123456789abcdef", 70<<10)), 0o644)
	os.WriteFile(filepath.Join(tpl, "vars.md"), []byte("# {{title}}\n{{fecha}} {{Date}}\n"), 0o644)
	before := snapshot(t, dir)
	for _, name := range []string{"bin", "utf16", "grande"} {
		out, err := run(t, dir, "note", "new", "Nota "+name, "--template", name)
		if err == nil || ExitCode(err) != 2 || out != "" {
			t.Errorf("--template %s: código %d, salida %q, error %v; se esperaba 2 sin salida", name, ExitCode(err), out, err)
		}
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("note new con una plantilla inválida no debe crear nada")
	}
	os.WriteFile(filepath.Join(tpl, "daily.md"), []byte("# x\x00"), 0o644)
	before = snapshot(t, dir)
	if out, err := run(t, dir, "daily"); err == nil || ExitCode(err) != 2 || out != "" {
		t.Errorf("daily con plantilla binaria: código %d, salida %q, error %v", ExitCode(err), out, err)
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("una plantilla inválida no debe crear nada (ni journal/)")
	}

	var errBuf bytes.Buffer
	old := Stderr
	Stderr = &errBuf
	defer func() { Stderr = old }()
	out, err := run(t, dir, "note", "new", "Con variables", "--template", "vars", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var n struct {
		Path     string   `json:"path"`
		Warnings []string `json:"warnings"`
	}
	if json.Unmarshal([]byte(out), &n) != nil || !reflect.DeepEqual(n.Warnings, []string{"{{fecha}}"}) {
		t.Errorf("warnings en el JSON: %q", out)
	}
	if !strings.Contains(errBuf.String(), "{{fecha}}") || strings.Contains(errBuf.String(), "{{Date}}") {
		t.Errorf("el aviso va por stderr y solo de la desconocida: %q", errBuf.String())
	}
	if b, _ := os.ReadFile(n.Path); !strings.HasPrefix(string(b), "# Con variables\n{{fecha}} 20") {
		t.Errorf("la nota: %q", b)
	}
}

// TestHostileFilesCLI (ORD-015 C.5 S2 y S3): `note list` con un FIFO llamado x.md no se cuelga y no lo lista; `note show` de una nota de más de
// 10 MB sale con 2 sin leerla; `task list` tampoco se cuelga.
func TestHostileFilesCLI(t *testing.T) {
	dir := fixture(t)
	if runtime.GOOS != "windows" {
		if err := mkfifo(filepath.Join(dir, "trampa.md")); err != nil {
			t.Fatal(err)
		}
	}
	big := filepath.Join(dir, "enorme.md")
	f, _ := os.Create(big)
	f.Truncate(storage.MaxNoteBytes + 1)
	f.Close()
	type res struct {
		out string
		err error
	}
	call := func(args ...string) res {
		ch := make(chan res, 1)
		go func() { o, e := run(t, dir, args...); ch <- res{o, e} }()
		select {
		case r := <-ch:
			return r
		case <-time.After(5 * time.Second):
			t.Fatalf("%v: se colgó", args)
			return res{}
		}
	}
	if r := call("note", "list"); r.err != nil || strings.Contains(r.out, "trampa") {
		t.Errorf("note list: %v %q", r.err, r.out)
	}
	if r := call("task", "list"); r.err != nil {
		t.Errorf("task list: %v", r.err)
	}
	if r := call("note", "show", big); r.err == nil || ExitCode(r.err) != 2 {
		t.Errorf("note show de una nota enorme: código %d, error %v", ExitCode(r.err), r.err)
	}
}

// setDateFormat escribe date_format en la configuración de la prueba (HOME aislado) para la carpeta de notas dir.
func setDateFormat(t *testing.T, dir, format string) {
	t.Helper()
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	c.DateFormat = format
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
}

// TestDatesMigrateCLI (ORD-017 F3): `dates migrate --dry-run` muestra el diff sin escribir; `migrate` pasa las fechas al otro formato; la segunda corrida no cambia
// nada; sin --to o con un valor inválido sale con 2 sin tocar nada.
func TestDatesMigrateCLI(t *testing.T) {
	dir := fixture(t)
	p := filepath.Join(dir, "fechas.md")
	orig := "# Fechas\n- [ ] uno 🛫 2026-05-01 📅 2026-05-10\n- [x] dos ✅ 2026-05-09\ntexto 📅 2026-05-10 suelto\n"
	os.WriteFile(p, []byte(orig), 0o644)
	before := snapshot(t, dir)
	out, err := run(t, dir, "dates", "migrate", "--to", "dataview", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"fechas.md:2", "- - [ ] uno 🛫 2026-05-01 📅 2026-05-10", "+ - [ ] uno [start:: 2026-05-01] [due:: 2026-05-10]", "+ - [x] dos [completion:: 2026-05-09]"} {
		if !strings.Contains(out, want) {
			t.Errorf("el dry-run debe mostrar %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "simulación") && !strings.Contains(out, "dry run") {
		t.Errorf("el dry-run lo dice: %s", out)
	}
	if strings.Contains(out, "suelto") {
		t.Error("un párrafo no es una tarea: no se toca")
	}
	if after := snapshot(t, dir); !reflect.DeepEqual(before, after) {
		t.Error("--dry-run no debe escribir nada")
	}
	if _, err := run(t, dir, "dates", "migrate", "--to", "dataview"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "# Fechas\n- [ ] uno [start:: 2026-05-01] [due:: 2026-05-10]\n- [x] dos [completion:: 2026-05-09]\ntexto 📅 2026-05-10 suelto\n" {
		t.Errorf("migrado:\n%s", b)
	}
	snap := snapshot(t, dir)
	again, err := run(t, dir, "dates", "migrate", "--to", "dataview")
	if err != nil || !(strings.Contains(again, "0 línea(s)") || strings.Contains(again, "0 line(s)")) || !reflect.DeepEqual(snap, snapshot(t, dir)) {
		t.Errorf("la segunda corrida no cambia nada: %v %q", err, again)
	}
	var js struct {
		To      string                         `json:"to"`
		DryRun  bool                           `json:"dry_run"`
		Lines   int                            `json:"lines"`
		Changes []struct{ Note, After string } `json:"changes"`
	}
	o2, err := run(t, dir, "dates", "migrate", "--to", "emoji", "--dry-run", "--json")
	if err != nil || json.Unmarshal([]byte(o2), &js) != nil || js.To != "emoji" || !js.DryRun || js.Lines != 2 {
		t.Errorf("--json: %v %q %+v", err, o2, js)
	}
	for _, args := range [][]string{{"dates", "migrate"}, {"dates", "migrate", "--to", "xml"}, {"dates", "otra"}} {
		snap := snapshot(t, dir)
		if _, err := run(t, dir, args...); err == nil || ExitCode(err) != 2 {
			t.Errorf("%v: se esperaba exit 2, dio %v", args, err)
		}
		if !reflect.DeepEqual(snap, snapshot(t, dir)) {
			t.Errorf("%v: no debe tocar nada", args)
		}
	}
}

// TestStartAfterDueWarningCLI (ORD-017 F5 / L8): `task start` / `task due` que dejan el inicio después del vencimiento avisan por stderr, sin cambiar el código de
// salida (0) ni la escritura; con el orden correcto no avisa.
func TestStartAfterDueWarningCLI(t *testing.T) {
	dir := fixture(t)
	id := taskID(t, dir, "Escribir informe", "proyecto.md")
	var errBuf bytes.Buffer
	old := Stderr
	Stderr = &errBuf
	defer func() { Stderr = old }()
	if _, err := run(t, dir, "task", "due", id, "2026-10-05"); err != nil {
		t.Fatal(err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("solo con vencimiento no hay aviso: %q", errBuf.String())
	}
	id = taskID(t, dir, "Escribir informe", "proyecto.md")
	if _, err := run(t, dir, "task", "start", id, "2026-10-08"); err != nil {
		t.Fatalf("avisa pero sale con 0: %v", err)
	}
	if e := errBuf.String(); !strings.Contains(e, "2026-10-08") || !strings.Contains(e, "2026-10-05") {
		t.Errorf("debe avisar con las dos fechas: %q", e)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "proyecto.md"))
	if !strings.Contains(string(b), "2026-10-08") {
		t.Errorf("el aviso no impide escribir: %s", b)
	}
	errBuf.Reset()
	id = taskID(t, dir, "Escribir informe", "proyecto.md")
	if _, err := run(t, dir, "task", "start", id, "2026-10-01"); err != nil || errBuf.Len() != 0 {
		t.Errorf("con el orden correcto no avisa: %v %q", err, errBuf.String())
	}
}

// TestDatesMigrateMidwayCLI (ORD-017 rev 2, R2-3): si una nota no se puede escribir, el CLI dice qué notas ya se migraron y sale con un código distinto de 0.
func TestDatesMigrateMidwayCLI(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("necesita una carpeta de solo lectura")
	}
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("- [ ] t 📅 2026-05-10\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "z"), 0o755)
	os.WriteFile(filepath.Join(dir, "z", "b.md"), []byte("- [ ] t 📅 2026-05-10\n"), 0o644)
	os.Chmod(filepath.Join(dir, "z"), 0o555)
	t.Cleanup(func() { os.Chmod(filepath.Join(dir, "z"), 0o755) })
	out, err := run(t, dir, "dates", "migrate", "--to", "dataview")
	if err == nil {
		t.Fatalf("debe fallar con la carpeta de solo lectura:\n%s", out)
	}
	if !strings.Contains(out, "a.md") || strings.Contains(out, "b.md") {
		t.Errorf("informa lo ya migrado (a.md) y no lo que falló:\n%s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.md")); string(b) != "- [ ] t [due:: 2026-05-10]\n" {
		t.Errorf("a.md quedó migrada: %q", b)
	}
}

// TestDatesMigrateFirstNoteFailsCLI (ORD-017 rev 3): si falla la primera nota, el mensaje es el error (sin "0 line(s) in 0 note(s)") y el código no es 0.
func TestDatesMigrateFirstNoteFailsCLI(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("necesita una carpeta de solo lectura")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a"), 0o755)
	os.WriteFile(filepath.Join(dir, "a", "n.md"), []byte("- [ ] t 📅 2026-05-10\n"), 0o644)
	os.Chmod(filepath.Join(dir, "a"), 0o555)
	t.Cleanup(func() { os.Chmod(filepath.Join(dir, "a"), 0o755) })
	out, err := run(t, dir, "dates", "migrate", "--to", "dataview")
	if err == nil {
		t.Fatalf("debe fallar:\n%s", out)
	}
	if strings.Contains(out, "0 line") || strings.Contains(out, "interrump") || strings.Contains(out, "interrupted") {
		t.Errorf("no debe parecer una migración hecha:\n%s", out)
	}
	if !strings.Contains(err.Error(), "n.md") {
		t.Errorf("el error nombra la nota: %v", err)
	}
}

// TestNoteIDIsRelativePathCLI (ORD-018 L13): `note list --json`, `note new --json` y `note show` usan como id la ruta relativa (a/igual.md y b/igual.md no chocan) y
// `note show <id>` la lee.
func TestNoteIDIsRelativePathCLI(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a/igual.md", "b/igual.md"} {
		os.MkdirAll(filepath.Join(dir, filepath.Dir(f)), 0o755)
		os.WriteFile(filepath.Join(dir, f), []byte("# Igual "+f+"\n"), 0o644)
	}
	os.MkdirAll(filepath.Join(dir, "trabajo"), 0o755)
	out, err := run(t, dir, "note", "new", "Lanzamiento", "--folder", "trabajo", "--empty", "--json")
	var nn struct {
		ID string `json:"id"`
	}
	if err != nil || json.Unmarshal([]byte(out), &nn) != nil || nn.ID != "trabajo/lanzamiento.md" {
		t.Fatalf("note new: %v %s", err, out)
	}
	out, err = run(t, dir, "note", "list", "--json")
	var list []struct {
		ID string `json:"id"`
	}
	if err != nil || json.Unmarshal([]byte(out), &list) != nil {
		t.Fatalf("note list: %v %s", err, out)
	}
	seen := map[string]bool{}
	for _, n := range list {
		if seen[n.ID] {
			t.Errorf("id repetido %s", n.ID)
		}
		seen[n.ID] = true
		show, err := run(t, dir, "note", "show", n.ID, "--json")
		var sn struct{ ID, Content string }
		if err != nil || json.Unmarshal([]byte(show), &sn) != nil || sn.ID != n.ID {
			t.Errorf("note show %s: %v %s", n.ID, err, show)
		}
	}
	if !seen["a/igual.md"] || !seen["b/igual.md"] {
		t.Errorf("ids: %v", seen)
	}
}

// TestDatesMigrateWithoutToTellsTheVaultFormat (ORD-018 D1): sin --to (y sin subcomando) el error sale en el idioma de la interfaz, dice cuántas tareas hay en cada
// formato y sugiere el comando.
func TestDatesMigrateWithoutToTellsTheVaultFormat(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "n.md"), []byte("- [ ] a 📅 2026-05-10\n- [ ] b 🛫 2026-05-01\n- [ ] c [due:: 2026-05-12]\n- [ ] d sin fecha\n"), 0o644)
	for _, c := range []struct {
		lang string
		want []string
	}{
		{"es", []string{"falta --to", "2 tarea(s) con emojis", "1 tarea(s) con Dataview", "lazymark dates migrate --to dataview --dry-run"}},
		{"en", []string{"missing --to", "2 task(s) with emoji", "1 task(s) with Dataview", "lazymark dates migrate --to dataview --dry-run"}},
	} {
		i18n.SetLanguage(c.lang)
		t.Cleanup(func() { i18n.SetLanguage("es") })
		_, err := run(t, dir, "dates", "migrate")
		if err == nil {
			t.Fatalf("%s: sin --to debe fallar", c.lang)
		}
		for _, w := range c.want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("%s: el error debe decir %q:\n%v", c.lang, w, err)
			}
		}
		if c.lang == "en" && strings.Contains(err.Error(), "falta") {
			t.Errorf("en inglés no hay español: %v", err)
		}
	}
	// un vault solo con Dataview sugiere emoji; sin tareas con fechas, lo dice
	os.WriteFile(filepath.Join(dir, "n.md"), []byte("- [ ] c [due:: 2026-05-12]\n"), 0o644)
	i18n.SetLanguage("en")
	_, err := run(t, dir, "dates", "migrate")
	if err == nil || !strings.Contains(err.Error(), "--to emoji --dry-run") {
		t.Errorf("solo Dataview → sugiere emoji: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "n.md"), []byte("- [ ] sin fecha\n"), 0o644)
	_, err = run(t, dir, "dates", "migrate")
	if err == nil || !strings.Contains(err.Error(), "No task has dates") {
		t.Errorf("sin fechas: %v", err)
	}
	_, err = run(t, dir, "dates")
	if err == nil || strings.Contains(err.Error(), "falta el subcomando") {
		t.Errorf("sin subcomando, en inglés: %v", err)
	}
}

// TestCLIErrorsFollowTheLanguage (ORD-019 C.7 / L14): los errores de la CLI salen en el idioma de la interfaz, como los avisos: tarea inexistente, regex costosa,
// ruta fuera de la carpeta, nota que cambió afuera y argumentos inválidos; y la configuración (language) manda sobre el idioma del sistema.
func TestCLIErrorsFollowTheLanguage(t *testing.T) {
	dir := fixture(t)
	defer i18n.SetLanguage("es")
	cases := []struct {
		name string
		args []string
		es   string
		en   string
	}{
		{"tarea inexistente", []string{"task", "toggle", "ideas.md#00000000"}, "no existe esa tarea", "no such task"},
		{"regex costosa", []string{"search", `\w{1000}x`, "--regex"}, "expresión regular demasiado costosa", "regular expression too costly"},
		{"ruta fuera", []string{"note", "show", "../../etc/passwd.md"}, "la ruta no es una nota de la carpeta de notas", "the path is not a note in the notes folder"},
		{"argumentos inválidos", []string{"task", "due", "ideas.md#00000000", "no-es-fecha"}, "no es una fecha válida", "is not a valid date"},
		{"columna inexistente", []string{"task", "move", "ideas.md#00000000", "cancelada"}, "no existe", "does not exist"},
	}
	for _, lang := range []string{"es", "en"} {
		i18n.SetLanguage(lang)
		for _, c := range cases {
			_, err := run(t, dir, c.args...)
			want := c.es
			other := c.en
			if lang == "en" {
				want, other = c.en, c.es
			}
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("%s (%s): el error debe decir %q: %v", c.name, lang, want, err)
			} else if strings.Contains(err.Error(), other) {
				t.Errorf("%s (%s): no mezcla idiomas (%q): %v", c.name, lang, other, err)
			}
		}
	}
}

// TestNoteChangedErrorFollowsTheLanguage: "la nota cambió por fuera" también sale en el idioma de la interfaz.
func TestNoteChangedErrorFollowsTheLanguage(t *testing.T) {
	defer i18n.SetLanguage("es")
	for lang, want := range map[string]string{"es": "la nota cambió por fuera", "en": "the note changed outside"} {
		i18n.SetLanguage(lang)
		if got := storage.ErrNoteChanged.Error(); !strings.Contains(got, want) {
			t.Errorf("%s: %q", lang, got)
		}
	}
}

// TestConfigLanguageBeatsSystem (ORD-019 C.7): con `language` explícito en la configuración, los errores de la CLI salen en ese idioma aunque el sistema diga otro.
func TestConfigLanguageBeatsSystem(t *testing.T) {
	dir := fixture(t)
	defer i18n.SetLanguage("es")
	i18n.SetLanguage("es") // el sistema
	base, err := os.UserConfigDir()
	if err != nil {
		t.Skip(err)
	}
	cfgDir := filepath.Join(base, "lazymark")
	os.MkdirAll(cfgDir, 0o755)
	cfgFile := filepath.Join(cfgDir, "config.json")
	if _, err := os.Stat(cfgFile); err == nil {
		t.Skip("ya hay una configuración en el HOME aislado")
	}
	os.WriteFile(cfgFile, []byte(`{"language":"en"}`), 0o644)
	t.Cleanup(func() { os.Remove(cfgFile) })
	_, err = run(t, dir, "task", "toggle", "ideas.md#00000000")
	if err == nil || !strings.Contains(err.Error(), "no such task") {
		t.Errorf("language=en en la configuración manda: %v", err)
	}
}

// TestTaskListColorOnlyOnTTY (ORD-020 K4): `task list` solo lleva color si la salida es una terminal; en una tubería (aquí un buffer), con --json o con NO_COLOR no hay
// ni un código ANSI. Con color, cada fecha lleva el de su estado.
func TestTaskListColorOnlyOnTTY(t *testing.T) {
	dir := fixture(t)
	os.WriteFile(filepath.Join(dir, "col.md"), []byte("# C\n- [ ] vencida 📅 2020-01-01\n- [ ] lejos 📅 2999-01-01\n"), 0o644)
	out, err := run(t, dir, "task", "list")
	if err != nil || strings.Contains(out, "\x1b") {
		t.Errorf("en una tubería no hay color: %v %q", err, out)
	}
	if out, err = run(t, dir, "task", "list", "--json"); err != nil || strings.Contains(out, "\x1b") {
		t.Errorf("--json no lleva color: %v", err)
	}
	var buf bytes.Buffer
	svc, _ := ops.New(dir)
	if colorEnabled(&buf, svc) {
		t.Error("un buffer no es una terminal")
	}
	line := datesSuffixColored(ops.TaskDTO{Due: "2020-01-01", Overdue: true}, "2026-10-06")
	if !strings.Contains(line, "\x1b[") || !strings.Contains(line, "2020-01-01") {
		t.Errorf("con color cada fecha lleva su código: %q", line)
	}
	if plainLine := datesSuffix(ops.TaskDTO{Due: "2020-01-01", Overdue: true}); strings.Contains(plainLine, "\x1b") {
		t.Errorf("datesSuffix sigue sin color: %q", plainLine)
	}
}

// TestTaskMoveColorOnlyOnTTY (ORD-024): la fecha de la línea que imprimen `task move`, `due`, `start` y `toggle` va en el color de su estado SOLO en una terminal, igual
// que en `task list`; en una tubería, con `--json` o con NO_COLOR no hay ni un código ANSI.
func TestTaskMoveColorOnlyOnTTY(t *testing.T) {
	dir := fixture(t)
	os.WriteFile(filepath.Join(dir, "col.md"), []byte("# C\n- [ ] vencida 📅 2020-01-01\n- [ ] lejos 📅 2999-01-01\n"), 0o644)
	idOf := func(text string) string {
		out, err := run(t, dir, "task", "list", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var tasks []struct{ ID, Text string }
		if err := json.Unmarshal([]byte(out), &tasks); err != nil {
			t.Fatal(err)
		}
		for _, k := range tasks {
			if k.Text == text {
				return k.ID
			}
		}
		t.Fatalf("no hay la tarea %q", text)
		return ""
	}
	moveTo := func(tty bool, args ...string) string { // corre el comando con un archivo real como salida (una terminal simulada o no)
		old := isTerminal
		isTerminal = func(uintptr) bool { return tty }
		t.Cleanup(func() { isTerminal = old })
		f, err := os.CreateTemp(t.TempDir(), "salida")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := RunTaskWithWriter(f, append(args[1:], "--dir", dir), dir); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(f.Name())
		return string(b)
	}
	id := idOf("vencida")
	if out := moveTo(true, "task", "move", id, "doing"); !strings.Contains(out, "\x1b[") || !strings.Contains(out, "2020-01-01") {
		t.Errorf("en una terminal la fecha vencida de task move lleva color: %q", out)
	}
	if out := moveTo(false, "task", "move", id, "todo"); strings.Contains(out, "\x1b") || !strings.Contains(out, "2020-01-01") {
		t.Errorf("sin terminal (tubería o archivo) no hay color: %q", out)
	}
	if out := moveTo(true, "task", "move", id, "doing", "--json"); strings.Contains(out, "\x1b") {
		t.Errorf("--json nunca lleva color: %q", out)
	}
	t.Setenv("NO_COLOR", "1")
	if out := moveTo(true, "task", "move", id, "todo"); strings.Contains(out, "\x1b") {
		t.Errorf("NO_COLOR apaga el color: %q", out)
	}
	t.Setenv("NO_COLOR", "")
	// los otros comandos que imprimen la misma línea: due, start y toggle
	later := idOf("lejos")
	for _, args := range [][]string{{"task", "due", later, "2999-02-02"}, {"task", "start", later, "2999-01-15"}, {"task", "toggle", later}} {
		if out := moveTo(true, args...); !strings.Contains(out, "\x1b[") {
			t.Errorf("%v en una terminal lleva color: %q", args[:2], out)
		}
		if out := moveTo(false, args...); strings.Contains(out, "\x1b") {
			t.Errorf("%v en una tubería no lleva color: %q", args[:2], out)
		}
	}
}
