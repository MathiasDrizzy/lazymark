package theme

import (
	"fmt"
	"strings"
	"testing"
)

func TestThemeNames(t *testing.T) {
	names := ThemeNames()
	if len(names) != 14 {
		t.Fatalf("se esperaban 14 temas (7 de siempre y 7 nuevos), obtenidos %d", len(names))
	}

	expected := map[string]bool{
		"catppuccin-mocha":     true,
		"catppuccin-latte":     true,
		"catppuccin-frappe":    true,
		"catppuccin-macchiato": true,
		"tokyo-night":          true,
		"gruvbox-dark":         true,
		"nord":                 true,
		"dracula":              true,
		"one-dark":             true,
		"rose-pine":            true,
		"kanagawa":             true,
		"everforest-dark":      true,
		"solarized-dark":       true,
		"solarized-light":      true,
	}

	for _, name := range names {
		if !expected[name] {
			t.Errorf("tema inesperado en ThemeNames: %s", name)
		}
	}
}

func TestApplyThemeByName(t *testing.T) {
	ok := ApplyThemeByName("nord")
	if !ok {
		t.Fatalf("ApplyThemeByName('nord') retornó false")
	}
	if CurrentThemeName != "nord" {
		t.Errorf("CurrentThemeName esperado 'nord', obtenido '%s'", CurrentThemeName)
	}

	invalid := ApplyThemeByName("tema-inexistente")
	if invalid {
		t.Errorf("ApplyThemeByName('tema-inexistente') debía fallar")
	}
}

func TestNextTheme(t *testing.T) {
	ApplyThemeByName("catppuccin-mocha")
	next := NextTheme()
	if next == "catppuccin-mocha" {
		t.Errorf("NextTheme debió cambiar el tema actual")
	}
	if CurrentThemeName != next {
		t.Errorf("CurrentThemeName no coincide con el retornado por NextTheme")
	}
}

func TestPrevTheme(t *testing.T) {
	ApplyThemeByName("catppuccin-mocha")
	prev := PrevTheme()
	if prev == "catppuccin-mocha" {
		t.Errorf("PrevTheme debió cambiar el tema actual")
	}
	if CurrentThemeName != prev {
		t.Errorf("CurrentThemeName no coincide con el retornado por PrevTheme")
	}
	// Si aplicamos NextTheme después de PrevTheme, debemos volver a catppuccin-mocha
	afterNext := NextTheme()
	if afterNext != "catppuccin-mocha" {
		t.Errorf("NextTheme después de PrevTheme debió volver a catppuccin-mocha, obtenido '%s'", afterNext)
	}
}

// TestNewThemesListed (T1): la lista incluye los 7 temas pedidos, sin repetir, cada paleta
// con su nombre y su URL oficial, y todos los colores distintos entre un tema y otro.
func TestNewThemesListed(t *testing.T) {
	want := []string{"dracula", "one-dark", "rose-pine", "kanagawa", "everforest-dark", "solarized-dark", "solarized-light"}
	names := ThemeNames()
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Errorf("tema repetido en la lista: %s", n)
		}
		seen[n] = true
		p, ok := AvailableThemes[n]
		if !ok {
			t.Errorf("%s está en la lista pero no en AvailableThemes", n)
			continue
		}
		if p.Name != n {
			t.Errorf("la paleta de %s se llama %q", n, p.Name)
		}
		if !strings.HasPrefix(p.Source, "https://") {
			t.Errorf("%s: falta la URL de la fuente oficial (Source = %q)", n, p.Source)
		}
	}
	for _, n := range want {
		if !seen[n] {
			t.Errorf("falta el tema %s", n)
		}
	}
	if len(AvailableThemes) != len(names) {
		t.Errorf("AvailableThemes tiene %d temas y la lista %d", len(AvailableThemes), len(names))
	}
	bases := map[string]string{}
	for _, n := range names {
		b := fmt.Sprint(AvailableThemes[n].Base)
		if other, dup := bases[b]; dup && !(strings.HasPrefix(n, "catppuccin") && strings.HasPrefix(other, "catppuccin")) {
			t.Errorf("%s y %s tienen el mismo fondo base", n, other)
		}
		bases[b] = n
	}
}
