package clipboard

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Saver extrae imágenes del portapapeles y las guarda en el directorio destino
type Saver struct {
	AssetsDir string
}

func New(assetsDir string) *Saver {
	return &Saver{AssetsDir: assetsDir}
}

// PasteImage guarda la imagen en el portapapeles con nombre timestamp.png y devuelve la ruta relativa
func (s *Saver) PasteImage(noteName string) (string, error) {
	fileName := fmt.Sprintf("%s-%d.png", noteName, time.Now().Unix())
	targetPath := filepath.Join(s.AssetsDir, fileName)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// Intentar pngpaste primero, luego fallback a osascript
		if path, err := exec.LookPath("pngpaste"); err == nil {
			cmd = exec.Command(path, targetPath)
		} else {
			// AppleScript nativo para extraer portapapeles a PNG sin herramientas extra
			script := fmt.Sprintf(`
				set targetFile to (POSIX file "%s")
				try
					set theData to the clipboard as «class PNGf»
					set f to open for access targetFile with write permission
					write theData to f
					close access f
				on error
					try
						close access targetFile
					end try
					error "No hay imagen en el portapapeles"
				end try
			`, targetPath)
			cmd = exec.Command("osascript", "-e", script)
		}
	case "linux":
		if path, err := exec.LookPath("wl-paste"); err == nil {
			cmd = exec.Command(path, "--type", "image/png", "--output", targetPath)
		} else if path, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("sh", "-c", fmt.Sprintf("%s -selection clipboard -t image/png -o > '%s'", path, targetPath))
		} else {
			return "", fmt.Errorf("se requiere 'wl-paste' o 'xclip' para pegar imágenes en Linux")
		}
	case "windows":
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			$img = [System.Windows.Forms.Clipboard]::GetImage()
			if ($img -ne $null) {
				$img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
			} else {
				exit 1
			}
		`, targetPath)
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psScript)
	default:
		return "", fmt.Errorf("sistema operativo no soportado: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error al pegar imagen: %v (%s)", err, string(output))
	}

	// Devolver la referencia relativa a Markdown
	return filepath.Join("assets", fileName), nil
}
