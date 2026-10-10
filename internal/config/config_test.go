package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// isolate apunta HOME y XDG_CONFIG_HOME a un directorio temporal para que
// Load y Save nunca toquen la configuración real.
func isolate(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
}

func writeDiskConfig(t *testing.T, json string) {
	t.Helper()
	p := configFilePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPersistsSettings(t *testing.T) {
	isolate(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConfirmDelete = false
	cfg.HideCompletedTasks = true
	cfg.TaskScope = "tag:trabajo"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Load(cfg.NotesDir)
	if err != nil {
		t.Fatal(err)
	}
	if again.ConfirmDelete {
		t.Error("ConfirmDelete=false no persistió al recargar")
	}
	if !again.HideCompletedTasks {
		t.Error("HideCompletedTasks=true no persistió al recargar")
	}
	if again.TaskScope != "tag:trabajo" {
		t.Errorf("TaskScope = %q", again.TaskScope)
	}
}

func TestLoadKeepsDefaultsForMissingFields(t *testing.T) {
	isolate(t)
	writeDiskConfig(t, `{"theme":"nord"}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "nord" {
		t.Errorf("Theme = %q", cfg.Theme)
	}
	if !cfg.ConfirmDelete || !cfg.MouseClick || cfg.TaskScope != "all" {
		t.Errorf("faltan defaults: confirm=%v mouse=%v scope=%q", cfg.ConfirmDelete, cfg.MouseClick, cfg.TaskScope)
	}
}

// Un config.json anterior al keymap de H1 trae settings "?" y cheatsheet "h";
// se migra a los atajos nuevos (? cheatsheet, "," settings).
func TestLoadMigratesOldKeymap(t *testing.T) {
	isolate(t)
	writeDiskConfig(t, `{"keybindings":{"settings":"?","cheatsheet":"h","new_note":"n"}}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Keybindings.Settings != "," || cfg.Keybindings.Cheatsheet != "?" {
		t.Errorf("keymap viejo no migrado: settings=%q cheatsheet=%q", cfg.Keybindings.Settings, cfg.Keybindings.Cheatsheet)
	}
	if cfg.KeymapVersion != KeymapVersion {
		t.Errorf("KeymapVersion = %d", cfg.KeymapVersion)
	}
}

func TestLoadKeepsCustomKeymapOfCurrentVersion(t *testing.T) {
	isolate(t)
	writeDiskConfig(t, `{"keymap_version":2,"keybindings":{"new_note":"n"}}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Keybindings.NewNote != "n" || cfg.Keybindings.Settings != "," {
		t.Errorf("new_note=%q settings=%q", cfg.Keybindings.NewNote, cfg.Keybindings.Settings)
	}
}

// TestLoadRespectsCustomEditor (H1-5): un editor configurado que existe se
// respeta tal cual, aunque no sea micro, vim, nano ni nvim.
func TestLoadRespectsCustomEditor(t *testing.T) {
	isolate(t)
	bin := filepath.Join(t.TempDir(), "mi-editor")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EDITOR", bin)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != bin {
		t.Errorf("Editor = %q, se esperaba %q", cfg.Editor, bin)
	}

	// el JSON se arma con json.Marshal: una ruta de Windows lleva barras invertidas
	raw, _ := json.Marshal(map[string]any{"keymap_version": 2, "editor": bin + " --wait"})
	writeDiskConfig(t, string(raw))
	cfg, err = Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Editor != bin+" --wait" {
		t.Errorf("Editor con argumentos = %q", cfg.Editor)
	}
}

func TestSplitEditor(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Program Files", "Mi Editor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "ed")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, in, wantBin, wantArgs string
	}{
		{"nombre simple", "micro", "micro", ""},
		{"nombre con argumentos", "code --wait -n", "code", "--wait -n"},
		{"ruta con espacios sin comillas", bin, bin, ""},
		{"ruta con espacios y argumentos", bin + " --wait", bin, "--wait"},
		{"ruta entre comillas dobles", `"` + bin + `" --wait`, bin, "--wait"},
		{"ruta entre comillas simples", "'" + bin + "'", bin, ""},
		{"espacios de más", "  vim  -u  NONE ", "vim", "-u NONE"},
		{"vacío", "   ", "", ""},
	}
	for _, c := range cases {
		gotBin, gotArgs := SplitEditor(c.in)
		if gotBin != c.wantBin || strings.Join(gotArgs, " ") != c.wantArgs {
			t.Errorf("%s: SplitEditor(%q) = %q %q, se esperaba %q %q", c.name, c.in, gotBin, gotArgs, c.wantBin, c.wantArgs)
		}
	}
	for _, in := range []string{bin, bin + " --wait", `"` + bin + `" -n`} {
		if !EditorExists(in) {
			t.Errorf("EditorExists(%q) debería ser verdadero", in)
		}
	}
	if EditorExists(filepath.Join(dir, "no-existe")) || EditorExists("") {
		t.Error("EditorExists debería ser falso para un editor inexistente o vacío")
	}
}

// TestPopupBackgroundSetting (C.4): los valores son los mismos que los de screen_background (theme | terminal);
// por defecto "terminal"; "theme" se conserva al recargar; el valor de antes, "none" (sin fondo), y cualquier otro
// se migran en silencio a "terminal".
func TestPopupBackgroundSetting(t *testing.T) {
	isolate(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PopupBackground != PopupBackgroundTheme {
		t.Fatalf("por defecto = %q, se esperaba theme (como la pantalla)", cfg.PopupBackground)
	}
	for _, v := range []string{PopupBackgroundTheme, PopupBackgroundTerminal} {
		cfg.PopupBackground = v
		if err := cfg.Save(); err != nil {
			t.Fatal(err)
		}
		if again, _ := Load(cfg.NotesDir); again.PopupBackground != v {
			t.Errorf("tras recargar = %q, se esperaba %q", again.PopupBackground, v)
		}
	}
	writeDiskConfig(t, `{"keymap_version":2,"popup_background":"none"}`)
	if got, _ := Load(t.TempDir()); got.PopupBackground != PopupBackgroundTerminal {
		t.Errorf(`"none" -> %q, se esperaba terminal`, got.PopupBackground)
	}
	for _, bad := range []string{`"rojo"`, `""`, `3`} {
		writeDiskConfig(t, `{"keymap_version":2,"popup_background":`+bad+`}`)
		if got, _ := Load(t.TempDir()); got.PopupBackground != PopupBackgroundTheme {
			t.Errorf("valor %s -> %q, se esperaba theme", bad, got.PopupBackground)
		}
	}
	writeDiskConfig(t, `{"keymap_version":2}`)
	if got, _ := Load(t.TempDir()); got.PopupBackground != PopupBackgroundTheme {
		t.Errorf("ausente -> %q, se esperaba theme", got.PopupBackground)
	}
}

// TestDirFlagIsNotPersisted: la carpeta que se pasa con --dir vale solo para esa
// ejecución; guardar un ajuste cualquiera no la convierte en la carpeta por
// defecto. Antes, abrir lazymark con --dir y cambiar el tema la cambiaba para siempre.
func TestDirFlagIsNotPersisted(t *testing.T) {
	isolate(t)
	saved, oneOff := t.TempDir(), t.TempDir()
	raw, _ := json.Marshal(map[string]any{"keymap_version": 2, "notes_dir": saved})
	writeDiskConfig(t, string(raw))

	cfg, err := Load(oneOff)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NotesDir != oneOff {
		t.Fatalf("con --dir se debe usar %q, hay %q", oneOff, cfg.NotesDir)
	}
	cfg.Theme = "nord"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if again.NotesDir != saved || again.Theme != "nord" {
		t.Errorf("tras guardar un ajuste, la carpeta por defecto debía seguir siendo %q (hay %q) y el tema nord (hay %q)", saved, again.NotesDir, again.Theme)
	}
}

// TestDirFlagWithoutSavedConfig: sin configuración previa, --dir tampoco se guarda.
func TestDirFlagWithoutSavedConfig(t *testing.T) {
	isolate(t)
	cfg, _ := Load(t.TempDir())
	cfg.Theme = "nord"
	_ = cfg.Save()
	again, _ := Load("")
	if again.NotesDir != DefaultNotesDir() {
		t.Errorf("la carpeta por defecto debía ser %q y es %q", DefaultNotesDir(), again.NotesDir)
	}
}

// TestSetNotesDirPersists: elegir una carpeta de forma explícita (Ajustes) sí se guarda,
// incluso si la ejecución empezó con --dir.
func TestSetNotesDirPersists(t *testing.T) {
	isolate(t)
	chosen := t.TempDir()
	cfg, _ := Load(t.TempDir())
	cfg.SetNotesDir(chosen)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(""); again.NotesDir != chosen {
		t.Errorf("la carpeta elegida no se guardó: %q", again.NotesDir)
	}
}

// TestKeybindingModeMigratesLazygit (K1): el modo "lazygit" de una config existente
// se lee como "lazy" (sin atajos Vim); "dual" sigue igual; lo ausente o desconocido
// vuelve al valor por defecto; y al guardar queda "lazy" en el archivo.
func TestKeybindingModeMigratesLazygit(t *testing.T) {
	cases := map[string]string{
		`"lazygit"`: KeybindingModeLazy,
		`"LazyGit"`: KeybindingModeLazy,
		`"lazy"`:    KeybindingModeLazy,
		`"dual"`:    KeybindingModeDual,
		`"otro"`:    KeybindingModeDual,
		`""`:        KeybindingModeDual,
		`3`:         KeybindingModeDual,
	}
	for raw, want := range cases {
		isolate(t)
		writeDiskConfig(t, `{"keymap_version":2,"keybinding_mode":`+raw+`}`)
		cfg, err := Load(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if cfg.KeybindingMode != want {
			t.Errorf("keybinding_mode %s -> %q, se esperaba %q", raw, cfg.KeybindingMode, want)
		}
	}

	isolate(t)
	writeDiskConfig(t, `{"keymap_version":2,"keybinding_mode":"lazygit"}`)
	cfg, _ := Load(t.TempDir())
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(configFilePath())
	if !strings.Contains(string(data), `"keybinding_mode": "lazy"`) || strings.Contains(strings.ToLower(string(data)), "lazygit") {
		t.Errorf("el archivo guardado debe decir lazy y no lazygit:\n%s", data)
	}
}

// TestScreenBackgroundSetting (T4): por defecto "theme"; "terminal" se guarda y se lee; un valor
// ausente o desconocido vuelve a "theme".
func TestScreenBackgroundSetting(t *testing.T) {
	isolate(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ScreenBackground != ScreenBackgroundTheme {
		t.Fatalf("por defecto = %q, se esperaba theme", cfg.ScreenBackground)
	}
	cfg.ScreenBackground = ScreenBackgroundTerminal
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(cfg.NotesDir); again.ScreenBackground != ScreenBackgroundTerminal {
		t.Errorf("tras recargar = %q, se esperaba terminal", again.ScreenBackground)
	}
	for _, bad := range []string{`"rojo"`, `""`, `3`} {
		writeDiskConfig(t, `{"keymap_version":2,"screen_background":`+bad+`}`)
		if got, _ := Load(t.TempDir()); got.ScreenBackground != ScreenBackgroundTheme {
			t.Errorf("valor %s -> %q, se esperaba theme", bad, got.ScreenBackground)
		}
	}
	writeDiskConfig(t, `{"keymap_version":2}`)
	if got, _ := Load(t.TempDir()); got.ScreenBackground != ScreenBackgroundTheme {
		t.Errorf("sin el campo -> %q, se esperaba theme", got.ScreenBackground)
	}
}

// TestResolveVersion: la versión sale de ldflags si el release la inyectó; si no (go install), del módulo
// que registra Go (sin la "v"); y sin ninguna de las dos es "dev". Antes un binario de `go install` decía
// siempre 0.1.0.
func TestResolveVersion(t *testing.T) {
	info := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	cases := []struct {
		name    string
		ldflags string
		info    *debug.BuildInfo
		want    string
	}{
		{"release con ldflags", "0.2.1", info("v0.2.1"), "0.2.1"},
		{"ldflags manda sobre el módulo", "1.0.0", info("v0.2.1"), "1.0.0"},
		{"go install de una etiqueta", "", info("v0.2.1"), "0.2.1"},
		{"go install de un commit", "", info("v0.2.2-0.20261002120000-abcdef123456"), "0.2.2-0.20261002120000-abcdef123456"},
		{"compilado desde el árbol", "", info("(devel)"), "dev"},
		{"sin información", "", nil, "dev"},
		{"módulo sin versión", "", info(""), "dev"},
	}
	for _, c := range cases {
		if got := resolveVersion(c.ldflags, c.info); got != c.want {
			t.Errorf("%s: %q, se esperaba %q", c.name, got, c.want)
		}
	}
}

// TestMascotSetting (M3): por defecto sí; "mascot": false se guarda y se lee; sin el campo, sí.
func TestMascotSetting(t *testing.T) {
	isolate(t)
	cfg, _ := Load(t.TempDir())
	if !cfg.Mascot {
		t.Fatal("la mascota debe estar activada por defecto")
	}
	cfg.Mascot = false
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(cfg.NotesDir); again.Mascot {
		t.Error("tras desactivarla y recargar debe seguir desactivada")
	}
	writeDiskConfig(t, `{"keymap_version":2}`)
	if got, _ := Load(t.TempDir()); !got.Mascot {
		t.Error("sin el campo la mascota debe estar activada")
	}
}

// TestKanbanColumns (H4-1): por defecto todo, doing y done; se configuran con ids y título libre o por idioma; una
// lista inválida (menos de 2, ids repetidos o mal escritos) vuelve a la de por defecto.
func TestKanbanColumns(t *testing.T) {
	isolate(t)
	cfg, _ := Load(t.TempDir())
	if got := strings.Join(cfg.KanbanIDs(), ","); got != "todo,doing,done" {
		t.Fatalf("por defecto = %s", got)
	}
	if es, en := cfg.KanbanColumns[1].DisplayTitle("es"), cfg.KanbanColumns[1].DisplayTitle("en"); es != "En progreso" || en != "In progress" {
		t.Errorf("títulos por idioma: %q %q", es, en)
	}
	writeDiskConfig(t, `{"keymap_version":2,"kanban_columns":[
		{"id":"backlog","title":"Pendientes"},
		{"id":"doing","titles":{"es":"En curso","en":"Doing"}},
		{"id":"review"},
		{"id":"done"}]}`)
	cfg, _ = Load(t.TempDir())
	if got := strings.Join(cfg.KanbanIDs(), ","); got != "backlog,doing,review,done" {
		t.Fatalf("configuradas = %s", got)
	}
	cols := cfg.KanbanColumns
	for i, want := range []string{"Pendientes", "En curso", "review", "Completado"} {
		if got := cols[i].DisplayTitle("es"); got != want {
			t.Errorf("título %d en español = %q, se esperaba %q", i, got, want)
		}
	}
	if cols[1].DisplayTitle("en") != "Doing" || cols[0].DisplayTitle("en") != "Pendientes" {
		t.Error("título por idioma y libre en inglés")
	}
	for name, bad := range map[string]string{
		"una sola":       `[{"id":"a"}]`,
		"repetidas":      `[{"id":"a"},{"id":"a"}]`,
		"mayúsculas":     `[{"id":"A"},{"id":"b"}]`,
		"con espacio":    `[{"id":"a b"},{"id":"c"}]`,
		"con barra":      `[{"id":"a/b"},{"id":"c"}]`,
		"vacío":          `[]`,
		"demasiadas (7)": `[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"},{"id":"e"},{"id":"f"},{"id":"g"}]`,
		"id vacío":       `[{"id":""},{"id":"b"}]`,
	} {
		writeDiskConfig(t, `{"keymap_version":2,"kanban_columns":`+bad+`}`)
		got, _ := Load(t.TempDir())
		if strings.Join(got.KanbanIDs(), ",") != "todo,doing,done" {
			t.Errorf("%s: debía volver a las de por defecto: %v", name, got.KanbanIDs())
		}
	}
	// y se guardan tal cual
	cfg.KanbanColumns = []KanbanColumn{{ID: "a"}, {ID: "b", Title: "B"}}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(cfg.NotesDir); strings.Join(again.KanbanIDs(), ",") != "a,b" {
		t.Errorf("tras guardar y recargar: %v", again.KanbanIDs())
	}
}

// TestKanbanCardsConfig (C.6): "cards" por defecto; "compact" se conserva; un valor ausente o desconocido vuelve a "cards".
func TestKanbanCardsConfig(t *testing.T) {
	isolate(t)
	cfg, _ := Load(t.TempDir())
	if cfg.KanbanCards != KanbanCardsRects {
		t.Fatalf("por defecto = %q", cfg.KanbanCards)
	}
	cfg.KanbanCards = KanbanCardsCompact
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(cfg.NotesDir); again.KanbanCards != KanbanCardsCompact {
		t.Errorf("tras recargar = %q", again.KanbanCards)
	}
	for _, bad := range []string{`"grande"`, `""`, `3`} {
		writeDiskConfig(t, `{"keymap_version":2,"kanban_cards":`+bad+`}`)
		if got, _ := Load(t.TempDir()); got.KanbanCards != KanbanCardsRects {
			t.Errorf("valor %s -> %q", bad, got.KanbanCards)
		}
	}
}

// TestLanguageConfig (C.5): el idioma de la config es auto o el código de uno soportado; pt-BR y zh_CN valen como pt y zh; un
// valor desconocido es auto.
func TestLanguageConfig(t *testing.T) {
	isolate(t)
	for in, want := range map[string]string{
		`"auto"`: "auto", `"AUTO"`: "auto", `"es"`: "es", `"en"`: "en", `"pt-BR"`: "pt", `"fr"`: "fr", `"de"`: "de", `"it"`: "it",
		`"ja"`: "ja", `"zh_CN"`: "zh", `"zh-CN"`: "zh", `"klingon"`: "auto", `""`: "auto", `3`: "auto",
	} {
		writeDiskConfig(t, `{"keymap_version":2,"language":`+in+`}`)
		if got, _ := Load(t.TempDir()); got.Language != want {
			t.Errorf("language %s -> %q, se esperaba %q", in, got.Language, want)
		}
	}
}

// TestNerdFontDefault (C.8): nerd_font es true por defecto y también en una config que no tiene la clave; false se conserva.
func TestNerdFontDefault(t *testing.T) {
	isolate(t)
	cfg, _ := Load(t.TempDir())
	if !cfg.NerdFont {
		t.Fatal("por defecto debe ser true")
	}
	writeDiskConfig(t, `{"keymap_version":2}`)
	if got, _ := Load(t.TempDir()); !got.NerdFont {
		t.Error("una config sin la clave debe quedar en true")
	}
	writeDiskConfig(t, `{"keymap_version":2,"nerd_font":false}`)
	if got, _ := Load(t.TempDir()); got.NerdFont {
		t.Error("false se conserva")
	}
}

// TestOptionsDefaultsAndValidation (ORD-025): las 8 opciones nuevas valen, por defecto, el comportamiento de siempre; un valor cambiado se respeta; uno inválido o fuera de
// rango vuelve al defecto sin romper la lectura.
func TestOptionsDefaultsAndValidation(t *testing.T) {
	d := DefaultConfig("")
	if !d.DateWarnings || !d.DateFormatNotice || d.ClickHintIdleSeconds != 20 || d.ClickHintShowSeconds != 15 || d.ClickHintEverySeconds != 60 || d.TrashDays != 20 ||
		d.DailyFolder != "journal" || d.DailyName != "YYYY-MM-DD" || d.TemplatesFolder != "templates" || d.KanbanTag != "kb" || d.NotesSort != "name" || d.TasksSort != "note" || len(d.DateGlyphs) != 0 {
		t.Errorf("los valores por defecto cambiaron: %+v", d)
	}
	load := func(js string) *Config {
		t.Helper()
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
		t.Setenv("AppData", filepath.Join(home, "AppData"))
		base, _ := os.UserConfigDir()
		os.MkdirAll(filepath.Join(base, "lazymark"), 0o755)
		os.WriteFile(filepath.Join(base, "lazymark", "config.json"), []byte(js), 0o644)
		cfg, err := Load(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	// ausente: los defectos
	if c := load(`{}`); c.TrashDays != 20 || c.DailyFolder != "journal" || c.KanbanTag != "kb" || !c.DateWarnings {
		t.Errorf("un archivo sin las claves da los defectos: %+v", c)
	}
	// cambiados
	c := load(`{"date_warnings":false,"date_format_notice":false,"click_hint_idle_seconds":30,"click_hint_show_seconds":10,"click_hint_every_seconds":120,"trash_days":0,
	"daily_folder":"diario/2026","daily_name":"AAAA.MM.DD","templates_folder":"moldes","kanban_tag":"board","notes_sort":"modified","tasks_sort":"due",
	"date_glyphs":{"due":"D","start":"→"}}`)
	if c.DateWarnings || c.DateFormatNotice || c.ClickHintIdleSeconds != 30 || c.ClickHintShowSeconds != 10 || c.ClickHintEverySeconds != 120 || c.TrashDays != 0 ||
		c.DailyFolder != "diario/2026" || c.DailyName != "AAAA.MM.DD" || c.TemplatesFolder != "moldes" || c.KanbanTag != "board" || c.NotesSort != "modified" || c.TasksSort != "due" ||
		c.DateGlyphs["due"] != "D" || c.DateGlyphs["start"] != "→" {
		t.Errorf("los valores cambiados se respetan: %+v", c)
	}
	// inválidos: vuelven al defecto
	c = load(`{"click_hint_idle_seconds":1,"click_hint_show_seconds":9999,"click_hint_every_seconds":-3,"trash_days":9999,"daily_folder":"../fuera","daily_name":"nota","templates_folder":"/abs",
	"kanban_tag":"1 mal!","notes_sort":"zzz","tasks_sort":"???","date_glyphs":{"due":"ab","start":"","done":"\u0007","scheduled":" ","created":"日"}}`)
	if c.ClickHintIdleSeconds != 20 || c.ClickHintShowSeconds != 15 || c.ClickHintEverySeconds != 60 || c.TrashDays != 20 || c.DailyFolder != "journal" || c.DailyName != "YYYY-MM-DD" ||
		c.TemplatesFolder != "templates" || c.KanbanTag != "kb" || c.NotesSort != "name" || c.TasksSort != "note" {
		t.Errorf("los valores inválidos vuelven al defecto: %+v", c)
	}
	if len(c.DateGlyphs) != 1 || c.DateGlyphs["created"] != "日" { // solo vale un carácter de ancho 1 o 2: "日" mide 2
		t.Errorf("date_glyphs: solo el glifo válido se queda: %v", c.DateGlyphs)
	}
	// la visibilidad "se va antes de volver a salir": show >= every vuelve a los dos defectos
	if c := load(`{"click_hint_show_seconds":100,"click_hint_every_seconds":50}`); c.ClickHintShowSeconds != 15 || c.ClickHintEverySeconds != 60 {
		t.Errorf("show >= every: %d %d", c.ClickHintShowSeconds, c.ClickHintEverySeconds)
	}
}

// TestCleanRelFolderAndDailyName: las rutas se validan dentro de la carpeta de notas; el nombre diario lleva año, mes y día.
func TestCleanRelFolderAndDailyName(t *testing.T) {
	for in, want := range map[string]string{"journal": "journal", "a/b": "a/b", "a\\b": "a/b", " diario ": "diario", "a/b/": "a/b", "año/mes": "año/mes"} {
		if got, ok := CleanRelFolder(in); !ok || got != want {
			t.Errorf("CleanRelFolder(%q) = %q %v, se esperaba %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"CON", "diario/nul", "COM1.txt", "lpt9", "", "/abs", "C:\\x", "C:x", "..", "a/../b", "../x", ".oculta", "a/.git", "assets", "a//b", "a/./b", "x:y", "a|b", "a\x00b", string(make([]byte, 101))} {
		if _, ok := CleanRelFolder(bad); ok {
			t.Errorf("CleanRelFolder(%q) debe rechazarse", bad)
		}
	}
	for _, ok := range []string{"YYYY-MM-DD", "AAAA-MM-DD", "DD.MM.YYYY", "diario YYYY MM DD"} {
		if !ValidDailyName(ok) {
			t.Errorf("%q es un nombre diario válido", ok)
		}
	}
	for _, bad := range []string{"", "nota", "YYYY-MM", "YYYY/MM/DD", "..YYYY-MM-DD", "YYYY-MM-DD\n", ".YYYY-MM-DD"} {
		if ValidDailyName(bad) {
			t.Errorf("%q no es un nombre diario válido", bad)
		}
	}
	if got := FormatDailyName("AAAA.MM.DD", time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)); got != "2026.10.06" {
		t.Errorf("FormatDailyName = %q", got)
	}
}

// writeUserConfig deja js como el config.json del usuario (en un HOME aislado) y devuelve su ruta.
func writeUserConfig(t *testing.T, js string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
	base, _ := os.UserConfigDir()
	os.MkdirAll(filepath.Join(base, "lazymark"), 0o755)
	p := filepath.Join(base, "lazymark", "config.json")
	os.WriteFile(p, []byte(js), 0o644)
	return p
}

// TestTolerantConfigWrongType (ORD-025 rev 2): un campo con el tipo equivocado vuelve a su defecto con un aviso y el RESTO del archivo se respeta; al guardar, no se pierde nada (tema,
// idioma y formato de fechas siguen en el archivo). Es el caso que borraba toda la configuración.
func TestTolerantConfigWrongType(t *testing.T) {
	p := writeUserConfig(t, `{"theme":"dracula","language":"es","trash_days":"20","date_format":"emoji"}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "dracula" || cfg.Language != "es" || cfg.DateFormat != "emoji" {
		t.Errorf("el resto del archivo se respeta: %q %q %q", cfg.Theme, cfg.Language, cfg.DateFormat)
	}
	if cfg.TrashDays != 20 {
		t.Errorf("el campo inválido vuelve a su defecto: %d", cfg.TrashDays)
	}
	w := strings.Join(cfg.Warnings(), "\n")
	if !strings.Contains(w, "trash_days") || !strings.Contains(w, "número entero") && !strings.Contains(w, "whole number") || strings.Contains(w, "unmarshal") || strings.Contains(w, "Go struct") {
		t.Errorf("avisa del campo y del tipo esperado, sin el error interno de Go (F2): %q", w)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	// N4 (ORD-026): el valor que no valía NO se pisa con el defecto: el archivo conserva lo que decía ("20", para que quien lo editó lo corrija), y el aviso sigue saliendo
	for _, want := range []string{`"theme": "dracula"`, `"language": "es"`, `"date_format": "emoji"`, `"trash_days": "20"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("tras guardar, el archivo conserva %s:\n%s", want, b)
		}
	}
	if _, err := os.Stat(p + ".bak"); err == nil {
		t.Error("un archivo que sí se leyó no necesita copia")
	}
}

// TestTolerantConfigBrokenJSON: un config.json que no es JSON no se pierde: se usan los defectos, se avisa, y antes de guardar encima el original queda en config.json.bak (una
// sola vez; el archivo nuevo es JSON válido).
func TestTolerantConfigBrokenJSON(t *testing.T) {
	broken := `{"theme":"nord", "language": "es",  ` // cortado a la mitad
	p := writeUserConfig(t, broken)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "catppuccin-mocha" || len(cfg.Warnings()) == 0 || !strings.Contains(strings.Join(cfg.Warnings(), " "), "config.json") {
		t.Errorf("defectos y aviso: %q %v", cfg.Theme, cfg.Warnings())
	}
	if b, _ := os.ReadFile(p); string(b) != broken {
		t.Fatal("leer no toca el archivo")
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p + ".bak"); string(b) != broken {
		t.Errorf("el original queda en config.json.bak: %q", b)
	}
	var back map[string]any
	if b, _ := os.ReadFile(p); json.Unmarshal(b, &back) != nil {
		t.Error("el archivo nuevo es JSON válido")
	}
	if !strings.Contains(strings.Join(cfg.Warnings(), " "), ".bak") {
		t.Errorf("avisa de dónde quedó la copia: %v", cfg.Warnings())
	}
	// una segunda vez no apila otra copia (el archivo ya se leyó bien)
	cfg.Theme = "nord"
	cfg.Save()
	if _, err := os.Stat(p + ".bak.1"); err == nil {
		t.Error("no se hace otra copia al guardar de nuevo")
	}
	// un config roto otra vez: no pisa la copia anterior
	os.WriteFile(p, []byte(`[1,2`), 0o644)
	cfg2, _ := Load(t.TempDir())
	cfg2.Save()
	if b, _ := os.ReadFile(p + ".bak"); string(b) != broken {
		t.Errorf("la copia anterior no se pisa: %q", b)
	}
	if b, _ := os.ReadFile(p + ".bak.1"); string(b) != `[1,2` {
		t.Errorf("el segundo roto va a .bak.1: %q", b)
	}
}

// TestTolerantConfigUnknownKey: una clave que lazymark no conoce se conserva al guardar (y se avisa de ella): un error de tecleo o una opción de una versión más nueva no se borran.
func TestTolerantConfigUnknownKey(t *testing.T) {
	p := writeUserConfig(t, `{"theme":"nord","mi_clave":{"a":[1,2,3]},"trash_day":5}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := strings.Join(cfg.Warnings(), "\n")
	if !strings.Contains(w, "mi_clave") || !strings.Contains(w, "trash_day") {
		t.Errorf("avisa de las claves desconocidas: %q", w)
	}
	cfg.Theme = "dracula"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	var back map[string]json.RawMessage
	b, _ := os.ReadFile(p)
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if string(back["mi_clave"]) != `{"a":[1,2,3]}` && !strings.Contains(strings.ReplaceAll(strings.ReplaceAll(string(back["mi_clave"]), " ", ""), "\n", ""), `{"a":[1,2,3]}`) {
		t.Errorf("mi_clave se conserva tal cual: %s", back["mi_clave"])
	}
	if string(back["trash_day"]) != "5" || !strings.Contains(string(b), `"theme": "dracula"`) {
		t.Errorf("la otra clave y el cambio de tema también: %s", b)
	}
	// y sigue leyéndose bien después
	again, _ := Load(t.TempDir())
	if again.Theme != "dracula" {
		t.Errorf("tema: %q", again.Theme)
	}
}

// TestTolerantConfigUnreadableFile: un config.json que existe pero no se puede leer (aquí, sin permisos) tampoco se pisa a ciegas: se avisa y, si hay que guardar, antes se intenta la
// copia (sin permiso de lectura no se puede copiar: entonces NO se escribe encima y Save devuelve el error).
func TestTolerantConfigUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("necesita permisos de archivo de Unix y no ser root")
	}
	p := writeUserConfig(t, `{"theme":"nord"}`)
	os.Chmod(p, 0)
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	cfg, err := Load(t.TempDir())
	if err != nil || len(cfg.Warnings()) == 0 {
		t.Fatalf("avisa: %v %v", err, cfg.Warnings())
	}
	if err := cfg.Save(); err == nil {
		t.Error("sin poder copiar el original, Save no escribe encima")
	}
	os.Chmod(p, 0o644)
	if b, _ := os.ReadFile(p); string(b) != `{"theme":"nord"}` {
		t.Errorf("el original sigue intacto: %q", b)
	}
}

// TestWrongTypeIsKeptUntilChanged (ORD-026 N4): un valor con tipo equivocado se conserva tal cual al guardar; si el usuario cambia ese campo (desde Ajustes), se guarda el suyo.
func TestWrongTypeIsKeptUntilChanged(t *testing.T) {
	p := writeUserConfig(t, `{"theme":"nord","trash_days":"veinte"}`)
	cfg, _ := Load(t.TempDir())
	cfg.Theme = "dracula" // otro campo: no afecta al inválido
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), `"trash_days": "veinte"`) || !strings.Contains(string(b), `"theme": "dracula"`) {
		t.Errorf("conserva el inválido y guarda el tema nuevo:\n%s", b)
	}
	cfg2, _ := Load(t.TempDir())
	cfg2.TrashDays = 7 // ahora el usuario sí lo cambia
	if err := cfg2.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), `"trash_days": 7`) {
		t.Errorf("lo que el usuario elige se guarda:\n%s", b)
	}
	if cfg3, _ := Load(t.TempDir()); len(cfg3.Warnings()) != 0 || cfg3.TrashDays != 7 {
		t.Errorf("y deja de avisar: %v", cfg3.Warnings())
	}
}

// TestOutOfRangeValueWarnsAndIsKept (ORD-026 N4): un valor fuera de rango que se normaliza avisa igual que uno de tipo equivocado y se conserva al guardar; las migraciones y los
// valores vacíos de versiones anteriores no avisan; y un config.json recién guardado se vuelve a leer sin avisos.
func TestOutOfRangeValueWarnsAndIsKept(t *testing.T) {
	p := writeUserConfig(t, `{"sidebar_ratio":9,"due_soon_days":500,"trash_days":99999,"keybinding_mode":"lazygit","popup_background":"none","screen_background":""}`)
	cfg, _ := Load(t.TempDir())
	w := strings.Join(cfg.Warnings(), "\n")
	for _, k := range []string{"sidebar_ratio", "due_soon_days", "trash_days"} {
		if !strings.Contains(w, k) {
			t.Errorf("avisa de %s fuera de rango:\n%s", k, w)
		}
	}
	for _, k := range []string{"keybinding_mode", "popup_background", "screen_background"} {
		if strings.Contains(w, k) {
			t.Errorf("%s es una migración o un vacío, no avisa:\n%s", k, w)
		}
	}
	if cfg.SidebarRatio >= 0.75 || cfg.DueSoonDays != 30 {
		t.Errorf("se normaliza: %v %d", cfg.SidebarRatio, cfg.DueSoonDays)
	}
	cfg.Save()
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), `"sidebar_ratio": 9`) || !strings.Contains(string(b), `"trash_days": 99999`) {
		t.Errorf("el archivo conserva lo que decía:\n%s", b)
	}
	// un archivo guardado desde cero se vuelve a leer sin avisos
	os.Remove(p)
	fresh, _ := Load(t.TempDir())
	if err := fresh.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(t.TempDir()); len(again.Warnings()) != 0 {
		t.Errorf("lo guardado por lazymark no avisa: %v", again.Warnings())
	}
}

// TestBackupNeverOverwritesOrFollowsLinks (ORD-026 N5): la copia de respaldo nunca pisa un .bak que ya existe ni escribe a través de un enlace simbólico; sin nombre libre no se
// escribe nada y se avisa.
func TestBackupNeverOverwritesOrFollowsLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("los enlaces simbólicos necesitan permisos especiales en Windows")
	}
	broken := `{"theme":`
	p := writeUserConfig(t, broken)
	victim := filepath.Join(t.TempDir(), "victima.txt")
	os.WriteFile(victim, []byte("no tocar"), 0o644)
	os.Symlink(victim, p+".bak")                           // .bak es un enlace a un archivo ajeno
	os.WriteFile(p+".bak.1", []byte("copia vieja"), 0o644) // y .bak.1 ya existe
	cfg, _ := Load(t.TempDir())
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(victim); string(b) != "no tocar" {
		t.Errorf("no se escribe a través del enlace: %q", b)
	}
	if b, _ := os.ReadFile(p + ".bak.1"); string(b) != "copia vieja" {
		t.Errorf("no se pisa una copia que ya existe: %q", b)
	}
	if b, _ := os.ReadFile(p + ".bak.2"); string(b) != broken {
		t.Errorf("la copia va al primer nombre libre (.bak.2): %q", b)
	}
	// sin nombre libre: no se escribe y se avisa
	p = writeUserConfig(t, broken)
	os.WriteFile(p+".bak", []byte("x"), 0o644)
	for i := 1; i <= 99; i++ {
		os.WriteFile(fmt.Sprintf("%s.bak.%d", p, i), []byte("x"), 0o644)
	}
	cfg, _ = Load(t.TempDir())
	if err := cfg.Save(); err == nil || !errors.Is(err, ErrNoBackupName) {
		t.Errorf("sin nombre libre Save falla con ErrNoBackupName: %v", err)
	}
	if b, _ := os.ReadFile(p); string(b) != broken {
		t.Errorf("y el original sigue intacto: %q", b)
	}
	if !strings.Contains(strings.Join(cfg.Warnings(), " "), "no se guardó") && !strings.Contains(strings.Join(cfg.Warnings(), " "), "not saved") {
		t.Errorf("y avisa: %v", cfg.Warnings())
	}
}

// TestRunOverridesAreNotSaved (ORD-026 F1): --no-mouse y --theme valen para esa ejecución y no se guardan en config.json; si el usuario cambia el tema desde Ajustes, ese sí se guarda.
func TestRunOverridesAreNotSaved(t *testing.T) {
	p := writeUserConfig(t, `{"theme":"nord","mouse_click":true}`)
	cfg, _ := Load(t.TempDir())
	cfg.OverrideMouse(false)
	cfg.OverrideTheme("dracula")
	if cfg.MouseClick || cfg.Theme != "dracula" {
		t.Fatalf("valen en esta ejecución: %v %q", cfg.MouseClick, cfg.Theme)
	}
	cfg.TrashDays = 5 // el usuario cambia otra cosa y se guarda
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), `"theme": "nord"`) || !strings.Contains(string(b), `"mouse_click": true`) || !strings.Contains(string(b), `"trash_days": 5`) {
		t.Errorf("no guarda las anulaciones y sí lo demás:\n%s", b)
	}
	cfg.Theme = "gruvbox" // elegido desde Ajustes
	cfg.Save()
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), `"theme": "gruvbox"`) {
		t.Errorf("el tema elegido en Ajustes sí se guarda:\n%s", b)
	}
}

// TestWarningsFollowInterfaceLanguage (ORD-026 F2): el texto de un aviso se escribe al mostrarlo, en el idioma de ese momento (el de la interfaz se fija después de leer el archivo).
func TestWarningsFollowInterfaceLanguage(t *testing.T) {
	writeUserConfig(t, `{"trash_days":"20"}`)
	old := i18n.CurrentLanguage()
	t.Cleanup(func() { i18n.SetLanguage(string(old)) })
	i18n.SetLanguage("en")
	cfg, _ := Load(t.TempDir())
	i18n.SetLanguage("es")
	if w := strings.Join(cfg.Warnings(), ""); !strings.Contains(w, "debe ser un número entero") {
		t.Errorf("en español: %q", w)
	}
	i18n.SetLanguage("en")
	if w := strings.Join(cfg.Warnings(), ""); !strings.Contains(w, "must be a whole number") {
		t.Errorf("en inglés: %q", w)
	}
}

// TestLoadReadsUserThemes: Load carga los temas de la carpeta themes junto a config.json, así el tema guardado
// puede ser uno de ellos, y un archivo de tema que no vale llega a los avisos.
func TestLoadReadsUserThemes(t *testing.T) {
	isolate(t)
	t.Cleanup(func() { theme.LoadUserThemes(t.TempDir()) })
	writeDiskConfig(t, `{"theme": "noche"}`)
	dir := ThemesDir()
	if filepath.Dir(dir) != filepath.Dir(configFilePath()) {
		t.Fatalf("la carpeta de temas %s no está junto a config.json", dir)
	}
	colors := `"base": "#0a1014", "mantle": "#070b0d", "surface0": "#162128", "surface1": "#222e36", "overlay0": "#758a96", "text": "#cccfd1", "subtext0": "#a9b0b5", "peach": "#99c1dc", "mauve": "#b389a7", "teal": "#5ba1a3", "green": "#7b9f7e", "red": "#bf878c", "blue": "#7f97be"`
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "noche.json"), []byte(`{`+colors+`, "yellow": "#a1966d"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "incompleto.json"), []byte(`{`+colors+`}`), 0o644)

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "noche" || !theme.ApplyThemeByName(cfg.Theme) {
		t.Errorf("el tema guardado noche no se pudo aplicar (Theme = %q)", cfg.Theme)
	}
	theme.ApplyThemeByName("catppuccin-mocha")
	if w := strings.Join(cfg.Warnings(), "\n"); !strings.Contains(w, "incompleto") || !strings.Contains(w, `"yellow"`) {
		t.Errorf("falta el aviso del tema incompleto: %q", w)
	}
	if p := strings.Join(cfg.Problems(), "\n"); strings.Contains(p, "incompleto") { // la CLI no lo repite en cada comando
		t.Errorf("el aviso de un tema llegó a Problems: %q", p)
	}
}
