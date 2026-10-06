package editors

import (
	"os"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// TestMain fija el idioma de los mensajes (los errores salen en el idioma de la interfaz): las pruebas comparan textos en español y no dependen del LANG de quien las corre.
// Las pruebas de idioma lo cambian ellas mismas.
func TestMain(m *testing.M) {
	i18n.SetLanguage("es")
	os.Exit(m.Run())
}
