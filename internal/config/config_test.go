package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// TestPopupBackgroundSetting (H1-11): por defecto "none"; "theme" se conserva al
// recargar y cualquier otro valor vuelve a "none".
func TestPopupBackgroundSetting(t *testing.T) {
	isolate(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PopupBackground != PopupBackgroundNone {
		t.Fatalf("por defecto = %q, se esperaba none", cfg.PopupBackground)
	}
	cfg.PopupBackground = PopupBackgroundTheme
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	if again, _ := Load(cfg.NotesDir); again.PopupBackground != PopupBackgroundTheme {
		t.Errorf("tras recargar = %q, se esperaba theme", again.PopupBackground)
	}
	for _, bad := range []string{`"rojo"`, `""`, `3`} {
		writeDiskConfig(t, `{"keymap_version":2,"popup_background":`+bad+`}`)
		if got, _ := Load(t.TempDir()); got.PopupBackground != PopupBackgroundNone {
			t.Errorf("valor %s -> %q, se esperaba none", bad, got.PopupBackground)
		}
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
