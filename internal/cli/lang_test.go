package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// TestMain fija el idioma de los mensajes (los errores salen en el idioma de la interfaz): las pruebas comparan textos en español y no dependen del LANG de quien las corre.
// Las pruebas de idioma lo cambian ellas mismas.
func TestMain(m *testing.M) {
	// una configuración de usuario aislada: ni el idioma ni nada de la config real de quien corre las pruebas entra en ellas
	home, err := os.MkdirTemp("", "lazymark-test-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	os.Setenv("AppData", filepath.Join(home, "AppData"))
	i18n.SetLanguage("es")
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
