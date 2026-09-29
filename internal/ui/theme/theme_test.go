package theme

import (
	"testing"
)

func TestThemeNames(t *testing.T) {
	names := ThemeNames()
	if len(names) != 7 {
		t.Fatalf("se esperaban 7 temas, obtenidos %d", len(names))
	}

	expected := map[string]bool{
		"catppuccin-mocha":     true,
		"catppuccin-latte":     true,
		"catppuccin-frappe":    true,
		"catppuccin-macchiato": true,
		"tokyo-night":          true,
		"gruvbox-dark":         true,
		"nord":                 true,
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
