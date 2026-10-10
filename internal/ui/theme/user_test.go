package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// colores de un tema de usuario completo, en el formato del archivo
const userThemeJSON = `{
  "source": "https://example.com/noche",
  "base": "#0a1014", "mantle": "#070b0d", "surface0": "#162128", "surface1": "#222e36",
  "overlay0": "#758a96", "text": "#cccfd1", "subtext0": "#a9b0b5", "peach": "#99c1dc",
  "mauve": "#b389a7", "teal": "#5ba1a3", "green": "#7b9f7e", "red": "#bf878c",
  "blue": "#7f97be", "yellow": "#a1966d"
}`

func writeTheme(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// problemText es el aviso en inglés, como lo vería el usuario
func problemText(p UserThemeProblem) string { return fmt.Sprintf(p.EN, p.Args...) }

func cleanUserThemes(t *testing.T) {
	t.Cleanup(func() {
		LoadUserThemes(t.TempDir())
		ApplyThemeByName("catppuccin-mocha")
	})
}

// TestUserThemeLoads: un archivo completo se carga con el nombre del archivo, va después de los temas
// incluidos y se aplica como cualquier otro, con sus colores.
func TestUserThemeLoads(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	writeTheme(t, dir, "noche.json", userThemeJSON)
	writeTheme(t, dir, "notas.txt", "no es un tema")

	if problems := LoadUserThemes(dir); len(problems) != 0 {
		t.Fatalf("avisos inesperados: %v", problems)
	}
	names := ThemeNames()
	if len(names) != 15 || names[14] != "noche" {
		t.Fatalf("se esperaba noche después de los 14 incluidos, lista: %v", names)
	}
	if !ApplyThemeByName("noche") || CurrentThemeName != "noche" {
		t.Fatalf("no se aplicó noche (actual %q)", CurrentThemeName)
	}
	if got := *hex(ColorPeach); got != "#99c1dc" {
		t.Errorf("Peach = %s, se esperaba #99c1dc", got)
	}
	if got := *hex(ColorBase); got != "#0a1014" {
		t.Errorf("Base = %s, se esperaba #0a1014", got)
	}
	if AvailableThemes["noche"].Source != "https://example.com/noche" {
		t.Errorf("Source = %q", AvailableThemes["noche"].Source)
	}
	if next := NextTheme(); next != "catppuccin-mocha" {
		t.Errorf("después del último tema NextTheme debe volver al primero, dio %q", next)
	}
}

// TestUserThemeReload: cargar otra carpeta quita los temas de la anterior, y una carpeta que no existe no avisa.
func TestUserThemeReload(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	writeTheme(t, dir, "noche.json", userThemeJSON)
	LoadUserThemes(dir)
	if problems := LoadUserThemes(filepath.Join(dir, "no-existe")); len(problems) != 0 {
		t.Errorf("una carpeta que no existe no debe avisar: %v", problems)
	}
	if _, ok := AvailableThemes["noche"]; ok || len(ThemeNames()) != 14 {
		t.Errorf("noche siguió cargado tras recargar: %v", ThemeNames())
	}
}

// TestUserThemeRejected: cada archivo que no vale se salta con un aviso que nombra el tema y el motivo, y
// no impide que se carguen los demás.
func TestUserThemeRejected(t *testing.T) {
	cleanUserThemes(t)
	cases := []struct{ file, body, want string }{
		{"falta.json", strings.Replace(userThemeJSON, `"peach": "#99c1dc",`, "", 1), `theme falta was not loaded: the color "peach" is missing`},
		{"corto.json", strings.Replace(userThemeJSON, `"#99c1dc"`, `"#9cd"`, 1), `theme corto was not loaded: the color "peach" is "#9cd", not #rrggbb`},
		{"numero.json", strings.Replace(userThemeJSON, `"#99c1dc"`, `12`, 1), `theme numero was not loaded: the color "peach" is 12, not #rrggbb`},
		{"errata.json", strings.Replace(userThemeJSON, `"base"`, `"bse"`, 1), `theme errata was not loaded: unknown key "bse"`},
		{"sobra.json", strings.Replace(userThemeJSON, `"base"`, `"extra": "#000000", "base"`, 1), `theme sobra was not loaded: unknown key "extra"`},
		{"roto.json", `{"base": `, `theme roto was not loaded: it is not a valid JSON object`},
		{"lista.json", `[]`, `theme lista was not loaded: it is not a valid JSON object`},
		{"nord.json", userThemeJSON, `theme nord was not loaded: a built-in theme has that name`},
		{"carpeta.json", "", `theme carpeta was not loaded: not a regular file`},
	}
	dir := t.TempDir()
	for _, c := range cases {
		if c.file == "carpeta.json" {
			if err := os.Mkdir(filepath.Join(dir, c.file), 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		writeTheme(t, dir, c.file, c.body)
	}
	writeTheme(t, dir, "valido.json", userThemeJSON)

	got := map[string]bool{}
	for _, p := range LoadUserThemes(dir) {
		got[problemText(p)] = true
	}
	for _, c := range cases {
		if !got[c.want] {
			t.Errorf("falta el aviso %q; avisos: %v", c.want, got)
		}
	}
	if len(got) != len(cases) {
		t.Errorf("se esperaban %d avisos, hubo %d: %v", len(cases), len(got), got)
	}
	if names := ThemeNames(); len(names) != 15 || names[14] != "valido" {
		t.Errorf("solo valido debía cargarse: %v", names)
	}
	if AvailableThemes["nord"].Source != Nord.Source {
		t.Error("un archivo llamado nord.json reemplazó el tema incluido")
	}
}

// loadOrUnblock carga dir. Si se cuelga en el FIFO, lo suelta y espera a que termine antes de fallar, para que no
// siga tocando los temas mientras corre otro test.
func loadOrUnblock(t *testing.T, dir, fifo string) []UserThemeProblem {
	t.Helper()
	done := make(chan []UserThemeProblem, 1)
	go func() { done <- LoadUserThemes(dir) }()
	select {
	case problems := <-done:
		return problems
	case <-time.After(5 * time.Second):
		releaseFIFO(fifo)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		t.Fatal("LoadUserThemes se colgó abriendo un FIFO")
		return nil
	}
}

// TestUserThemeFIFO: un FIFO llamado x.json, o una carpeta de temas que es un FIFO, da un aviso en vez de colgar
// el arranque esperando a quien escriba. (La carpeta no cuelga porque os.ReadDir abre con O_DIRECTORY; el test
// cuida que siga así.)
func TestUserThemeFIFO(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	fifo := filepath.Join(dir, "trampa.json")
	if err := mkfifo(fifo); err != nil {
		t.Skip("no se puede crear un FIFO:", err)
	}
	problems := loadOrUnblock(t, dir, fifo)
	if len(problems) != 1 || problemText(problems[0]) != "theme trampa was not loaded: not a regular file" {
		t.Errorf("FIFO en la carpeta, avisos: %v", problems)
	}

	themes := filepath.Join(t.TempDir(), "themes")
	if err := mkfifo(themes); err != nil {
		t.Fatal(err)
	}
	problems = loadOrUnblock(t, themes, themes)
	if len(problems) != 1 || !strings.HasPrefix(problemText(problems[0]), "could not read the themes folder "+themes) {
		t.Errorf("la carpeta es un FIFO, avisos: %v", problems)
	}
}

// TestUserThemeNamesRestricted: el nombre del archivo es el nombre del tema y sale en la ayuda de --theme y en la barra de estado; con
// comas o espacios parecería más de un tema, y con caracteres de control podría escribir secuencias de escape en la terminal. Solo valen
// [a-z0-9._-]; los demás se saltan con un aviso que no repite el nombre sin escapar.
func TestUserThemeNamesRestricted(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	bad := []string{"my theme, x.json", "Noche.json", "a\x1b[31mb.json", "a‮b.json", "a\u009db.json", "ñandú.json"}
	for _, name := range bad {
		writeTheme(t, dir, name, userThemeJSON)
	}
	writeTheme(t, dir, "noche-2.v1_x.json", userThemeJSON)
	problems := LoadUserThemes(dir)
	if len(problems) != len(bad) {
		t.Fatalf("avisos: %d, se esperaban %d: %v", len(problems), len(bad), problems)
	}
	for _, p := range problems {
		for _, text := range []string{problemText(p), fmt.Sprintf(p.ES, p.Args...)} {
			for _, r := range text {
				if r < 0x20 || (r >= 0x7f && r <= 0x9f) || r == 0x202e {
					t.Errorf("el aviso lleva el carácter de control %U sin escapar: %q", r, text)
				}
			}
		}
	}
	if got := strings.Join(userNames, ","); got != "noche-2.v1_x" {
		t.Errorf("temas cargados %q, se esperaba solo noche-2.v1_x", got)
	}
}

// TestUserThemeColorWarningEscapes: el valor de un color mal escrito se muestra tal cual en el aviso; sus caracteres de control van escapados.
func TestUserThemeColorWarningEscapes(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	body := strings.Replace(userThemeJSON, `"#0a1014"`, `"\u001b[2J\u009d‮"`, 1)
	writeTheme(t, dir, "malo.json", body)
	problems := LoadUserThemes(dir)
	if len(problems) != 1 {
		t.Fatalf("avisos: %v", problems)
	}
	for _, r := range problemText(problems[0]) {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) || r == 0x202e {
			t.Errorf("el aviso lleva el carácter de control %U sin escapar: %q", r, problemText(problems[0]))
		}
	}
}
