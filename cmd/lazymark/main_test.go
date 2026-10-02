package main

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// TestUsageIsAccurate: la ayuda de la línea de comandos ya no nombra teclas que
// no existen (antes anunciaba `p` para pegar y `t` para el tema) y apunta a ? y ,.
func TestUsageIsAccurate(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		i18n.SetLanguage(lang)
		text := usageHeader() + usageFooter()
		for _, stale := range []string{"[p]", "[t]", "[Espacio/x]", "[1..4]", "Pegar imagen", "Paste image", "--pending]           "} {
			if strings.Contains(text, stale) {
				t.Errorf("%s: la ayuda conserva texto obsoleto %q", lang, stale)
			}
		}
		for _, want := range []string{"task list", "task toggle", "task move", "note list", "note show", "note new", "mcp", "?", ","} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: a la ayuda le falta %q", lang, want)
			}
		}
	}
	i18n.SetLanguage("es")
}

// TestHelpIsLazyNotLazygit (L2): lazymark es "lazy", no "estilo lazygit": la ayuda, en español y en inglés, no
// menciona lazygit.
func TestHelpIsLazyNotLazygit(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		i18n.SetLanguage(lang)
		if text := usageHeader() + usageFooter(); strings.Contains(strings.ToLower(text), "lazygit") {
			t.Errorf("%s: la ayuda menciona lazygit:\n%s", lang, text)
		}
	}
}
