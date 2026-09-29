package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Client gestiona la emisión de comandos gráficos Kitty
type Client struct {
	Supported bool
}

// New detecta si la terminal actual soporta Kitty Graphics Protocol (Ghostty, Kitty, WezTerm)
func New() *Client {
	term := strings.ToLower(os.Getenv("TERM"))
	termProg := strings.ToLower(os.Getenv("TERM_PROGRAM"))

	// Ghostty, Kitty y WezTerm soportan el protocolo en macOS, Linux y Windows
	supported := strings.Contains(term, "kitty") ||
		strings.Contains(termProg, "ghostty") ||
		strings.Contains(termProg, "wezterm") ||
		strings.Contains(termProg, "kitty")

	return &Client{Supported: supported}
}

// RenderCommand genera la secuencia de escape para mostrar una imagen en disco
// t=f: transferencia por ruta de archivo local
// a=T: transmitir y presentar de inmediato
func (c *Client) RenderCommand(imagePath string, cols, rows int) string {
	if !c.Supported {
		return fmt.Sprintf("[🖼️ Imagen: %s (Terminal sin soporte gráfico Kitty)]", filepath.Base(imagePath))
	}

	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		absPath = imagePath
	}

	b64Path := base64.StdEncoding.EncodeToString([]byte(absPath))
	// a=T (transmit and display), f=100 (autodetect format), t=f (file path payload)
	// c=cols, r=rows (dimensionar)
	return fmt.Sprintf("\x1b_Ga=T,f=100,t=f,c=%d,r=%d;%s\x1b\\", cols, rows, b64Path)
}

// ClearAllCommand genera el escape para limpiar todas las imágenes activas en el terminal
func (c *Client) ClearAllCommand() string {
	if !c.Supported {
		return ""
	}
	return "\x1b_Ga=d\x1b\\"
}
