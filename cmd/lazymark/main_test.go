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
		for _, want := range []string{"task list", "task toggle", "note list", "note get", "mcp", "?", ","} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: a la ayuda le falta %q", lang, want)
			}
		}
	}
	i18n.SetLanguage("es")
}
