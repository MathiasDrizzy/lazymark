package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
			// Wayland: redirigir stdout al archivo destino
			cmd = exec.Command(path, "--type", "image/png")
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", fmt.Errorf("error al crear archivo destino: %v", err)
			}
			cmd.Stdout = outFile
			if err := cmd.Run(); err != nil {
				_ = outFile.Close()
				_ = os.Remove(targetPath)
				return "", fmt.Errorf("error al pegar imagen (wl-paste): %v", err)
			}
			_ = outFile.Close()
			// Devolver ruta normalizada para Markdown (siempre forward slashes)
			return filepath.ToSlash(filepath.Join("assets", fileName)), nil
		} else if path, err := exec.LookPath("xclip"); err == nil {
			// X11: redirigir stdout al archivo destino
			cmd = exec.Command(path, "-selection", "clipboard", "-t", "image/png", "-o")
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", fmt.Errorf("error al crear archivo destino: %v", err)
			}
			cmd.Stdout = outFile
			if err := cmd.Run(); err != nil {
				_ = outFile.Close()
				_ = os.Remove(targetPath)
				return "", fmt.Errorf("error al pegar imagen (xclip): %v", err)
			}
			_ = outFile.Close()
			return filepath.ToSlash(filepath.Join("assets", fileName)), nil
		} else {
			return "", fmt.Errorf("se requiere 'wl-paste' o 'xclip' para pegar imágenes en Linux")
		}
	case "windows":
		// Escapar comillas simples en ruta para PowerShell
		escapedPath := strings.ReplaceAll(targetPath, "'", "''")
		psScript := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			Add-Type -AssemblyName System.Drawing
			$img = [System.Windows.Forms.Clipboard]::GetImage()
			if ($img -ne $null) {
				$img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png)
			} else {
				exit 1
			}
		`, escapedPath)
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	default:
		return "", fmt.Errorf("sistema operativo no soportado: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error al pegar imagen: %v (%s)", err, string(output))
	}

	// Devolver la referencia relativa a Markdown (siempre forward slashes para compatibilidad)
	return filepath.ToSlash(filepath.Join("assets", fileName)), nil
}
