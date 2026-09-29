package config

import (
	"os"
	"path/filepath"
)

const (
	Version   = "0.1.0"
	AppName   = "lazymark"
)

// Config almacena las preferencias de ejecución de la aplicación.
type Config struct {
	NotesDir   string
	Editor     string
	MouseClick bool
	Theme      string
}

// Load carga la configuración por defecto o desde variables de entorno / rutas del usuario.
func Load(customDir string) (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	notesDir := customDir
	if notesDir == "" {
		notesDir = filepath.Join(home, "Documents", "notes")
	}

	// Asegurar que el directorio de notas exista
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, err
	}

	// Asegurar subdirectorio de assets / imágenes
	assetsDir := filepath.Join(notesDir, "assets")
	_ = os.MkdirAll(assetsDir, 0755)

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "micro" // Editor predeterminado de nuestro ecosistema
	}

	return &Config{
		NotesDir:   notesDir,
		Editor:     editor,
		MouseClick: true,
		Theme:      "catppuccin-mocha",
	}, nil
}
