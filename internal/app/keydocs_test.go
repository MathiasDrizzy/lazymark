package app

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/config"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// -update reescribe docs/keybindings.md y la tabla de atajos del README desde el
// keymap real: go test ./internal/app -run 'TestKeybindingsDoc|TestReadmeKeys' -update
var update = flag.Bool("update", false, "regenera docs/keybindings.md y la tabla del README")

// docKey escribe una tecla como código ("space" -> Space, "ctrl+v" -> Ctrl+V).
func docKey(k string) string {
	named := map[string]string{"space": "Space", "tab": "Tab", "shift+tab": "Shift+Tab", "enter": "Enter", "esc": "Esc",
		"home": "Home", "end": "End", "pgup": "PgUp", "pgdown": "PgDn"}
	switch {
	case named[k] != "":
		k = named[k]
	case strings.HasPrefix(k, "ctrl+"):
		k = "Ctrl+" + strings.ToUpper(strings.TrimPrefix(k, "ctrl+"))
	default:
		k = keyLabel(k)
	}
	return "`" + k + "`"
}

func docKeys(b Binding) string {
	keys := make([]string, len(b.Keys))
	for i, k := range b.Keys {
		keys[i] = docKey(k)
	}
	return strings.Join(keys, " ")
}

// defaultKeymap es el keymap por defecto (modo "dual": con la compatibilidad Vim).
func defaultKeymap() Keymap { return NewKeymap(config.DefaultConfig("")) }

// inEnglish ejecuta f con la interfaz en inglés (los documentos están en inglés).
func inEnglish(f func()) {
	prev := i18n.CurrentLanguage()
	i18n.SetLanguage("en")
	defer i18n.SetLanguage(string(prev))
	f()
}

var keySections = []struct {
	ctx   Context
	title string
	intro string
}{
	{ctxGlobal, "Global", "Work everywhere."},
	{ctxNav, "Navigation", "Move around lists, the preview and popups."},
	{ctxNotes, "Notes panel", "Panel `[1]`: the tree of folders and notes."},
	{ctxTasks, "Tasks panel", "Panel `[2]`: the checkboxes found in your notes. The preview follows the selected task."},
	{ctxTags, "Categories panel", "Panel `[3]`: the `#tags` of your notes."},
	{ctxPreview, "Preview panel", "Panel `[4]`: the rendered note."},
	{ctxKanban, "Kanban board", "Opened with `W`. Cards are the tasks of your notes."},
	{ctxPopup, "Lists inside popups", "Settings, move and the folder picker."},
	{ctxTrash, "Trash popup", "Opened with `x`."},
	{ctxConfirm, "Confirmation popups", ""},
}

// keybindingsDoc genera docs/keybindings.md desde el keymap.
func keybindingsDoc(keys Keymap) string {
	var b strings.Builder
	b.WriteString("# Keybindings\n\n")
	b.WriteString("<!-- Generated from the keymap by `go test ./internal/app -run TestKeybindingsDoc -update`. Do not edit by hand. -->\n\n")
	b.WriteString("Press `?` inside lazymark to see the keys of the panel you are in. The same list is generated from the same keymap, so it never disagrees with this page.\n\n")
	b.WriteString("`h` `j` `k` `l` `g` `G` `Ctrl+U` and `Ctrl+D` are Vim-style shortcuts. They work by default and can be turned off in Settings (Keybindings: `Lazy`). Nothing in lazymark needs Vim modes.\n\n")
	for _, sec := range keySections {
		bs := keys.In(sec.ctx)
		if len(bs) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", sec.title)
		if sec.intro != "" {
			b.WriteString(sec.intro + "\n\n")
		}
		b.WriteString("| Keys | Action |\n|---|---|\n")
		for _, bind := range bs {
			fmt.Fprintf(&b, "| %s | %s |\n", docKeys(bind), bind.Desc())
		}
		b.WriteString("\n")
	}
	b.WriteString(`## Mouse

- Click a note, task, tag, popup row or footer hint to select or run it.
- Click a task checkbox to toggle it. Click a tag to filter the notes tree.
- Drag the divider between the two columns to resize them.
- The scroll wheel scrolls the panel under the pointer.
- In popups the first click selects a row and the second click on a setting changes it. A click outside a popup does nothing; ` + "`Esc`" + ` closes it.

## Pasting images

- ` + "`Ctrl+V`" + ` saves the image on your clipboard into the ` + "`assets/`" + ` folder next to the note and adds ` + "`![](assets/…)`" + ` at the end of the note (or below the selected task when the Tasks panel is focused).
- Pasting a copied image file (for example with ` + "`Cmd+V`" + ` after copying it in Finder) imports that file the same way.

## Changing keys

Some actions can be rebound in the ` + "`keybindings`" + ` section of the configuration file. See [configuration.md](configuration.md).
`)
	return b.String()
}

// essentialRows son las filas del README (máximo 10): qué acciones y, opcionalmente, la descripción.
var essentialRows = []struct {
	ctx  Context
	acts []Action
	desc string // vacío: las descripciones del keymap unidas con " / "
}{
	{ctxNav, []Action{actUp, actDown}, ""},
	{ctxGlobal, []Action{actNextPanel}, ""},
	{ctxGlobal, []Action{actPanelNotes, actPanelTasks, actPanelTags, actPanelPreview}, "Jump to a panel"},
	{ctxNotes, []Action{actNewNote, actNewFolder}, ""},
	{ctxNotes, []Action{actRename, actMove, actDelete}, ""},
	{ctxTasks, []Action{actToggleTask}, "Toggle a task (Tasks panel)"},
	{ctxGlobal, []Action{actKanban}, ""},
	{ctxGlobal, []Action{actTrash}, ""},
	{ctxGlobal, []Action{actCheatsheet, actSettings}, ""},
	{ctxGlobal, []Action{actQuit}, ""},
}

// essentialTable genera la tabla de atajos esenciales del README desde el keymap.
func essentialTable(t *testing.T, keys Keymap) string {
	var b strings.Builder
	b.WriteString("| Keys | Action |\n|---|---|\n")
	for _, row := range essentialRows {
		var ks, ds []string
		for _, a := range row.acts {
			var found *Binding
			for _, bind := range keys.In(row.ctx) {
				if bind.Action == a {
					bind := bind
					found = &bind
					break
				}
			}
			if found == nil {
				t.Fatalf("la acción %d no está en el keymap del contexto %d", a, row.ctx)
			}
			ks = append(ks, docKey(found.Keys[0]))
			ds = append(ds, found.Desc())
		}
		desc := row.desc
		if desc == "" {
			desc = strings.Join(ds, " / ")
		}
		fmt.Fprintf(&b, "| %s | %s |\n", strings.Join(ks, " "), desc)
	}
	return b.String()
}

func repoFile(rel string) string { return filepath.Join("..", "..", filepath.FromSlash(rel)) }

// TestKeybindingsDoc: docs/keybindings.md es exactamente lo que genera el keymap.
func TestKeybindingsDoc(t *testing.T) {
	var want string
	inEnglish(func() { want = keybindingsDoc(defaultKeymap()) })
	path := repoFile("docs/keybindings.md")
	if *update {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("falta docs/keybindings.md (genéralo con -update): %v", err)
	}
	if string(got) != want {
		t.Errorf("docs/keybindings.md no coincide con el keymap; regenéralo con:\n  go test ./internal/app -run TestKeybindingsDoc -update")
	}
}

const (
	keysStart = "<!-- keys:start -->"
	keysEnd   = "<!-- keys:end -->"
)

// TestReadmeKeysTable (C6): la tabla de atajos del README coincide con el keymap
// real y no pasa de 10 filas.
func TestReadmeKeysTable(t *testing.T) {
	if len(essentialRows) > 10 {
		t.Fatalf("la tabla del README tiene %d filas (máximo 10)", len(essentialRows))
	}
	var want string
	inEnglish(func() { want = essentialTable(t, defaultKeymap()) })
	path := repoFile("README.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	i, j := strings.Index(text, keysStart), strings.Index(text, keysEnd)
	if i < 0 || j < i {
		t.Fatalf("el README no tiene los marcadores %s … %s", keysStart, keysEnd)
	}
	got := strings.TrimSpace(text[i+len(keysStart) : j])
	if *update {
		out := text[:i+len(keysStart)] + "\n" + want + text[j:]
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if got != strings.TrimSpace(want) {
		t.Errorf("la tabla de atajos del README no coincide con el keymap.\nREADME:\n%s\nkeymap:\n%s\nRegenérala con: go test ./internal/app -run TestReadmeKeys -update", got, want)
	}
}
