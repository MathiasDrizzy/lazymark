package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
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
