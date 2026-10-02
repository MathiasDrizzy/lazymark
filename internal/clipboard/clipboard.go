// Package clipboard incorpora imágenes a una nota: leyéndolas del portapapeles
// del sistema (Ctrl+V) o copiando un archivo de imagen cuya ruta llegó como
// texto pegado (Cmd+V sobre un archivo copiado en el Finder).
package clipboard

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Reader obtiene la imagen del portapapeles del sistema y la escribe como PNG
// en dest. Va detrás de una interfaz para que los tests nunca toquen el
// portapapeles real.
type Reader interface {
	ReadImage(dest string) error
}

// FileReader obtiene las rutas de los archivos copiados en el portapapeles (Cmd+C
// sobre un archivo en el Finder). También va detrás de una interfaz por R17.
type FileReader interface {
	ReadFiles() ([]string, error)
}

// ErrNoImage es lo que devuelve Paste cuando el portapapeles no tiene una imagen.
var ErrNoImage = errors.New("no hay una imagen copiada (ni una captura ni un archivo de imagen)")

// Saver guarda imágenes en la carpeta assets/ que hay junto a una nota.
type Saver struct {
	Reader Reader
	Files  FileReader // opcional: sin él, Paste solo mira las capturas
	now    func() time.Time
}

// New crea un Saver que lee el portapapeles real del sistema.
func New() *Saver {
	return &Saver{Reader: SystemReader{}, Files: SystemReader{}, now: time.Now}
}

// Paste guarda en <noteDir>/assets/ la imagen que haya copiada y devuelve su
// referencia relativa. Primero mira si hay un archivo de imagen copiado (se copia;
// el original no se toca) y después una captura (datos de imagen). Si no hay nada,
// devuelve ErrNoImage y no deja ninguna carpeta assets/ creada de más.
func (s *Saver) Paste(noteDir, noteName string) (string, error) {
	_, statErr := os.Stat(filepath.Join(noteDir, "assets"))
	existed := statErr == nil
	cleanup := func() {
		if !existed {
			_ = os.Remove(filepath.Join(noteDir, "assets")) // solo si quedó vacía
		}
	}
	if s.Files != nil {
		if paths, err := s.Files.ReadFiles(); err == nil {
			for _, p := range paths {
				if !isImageFile(p) {
					continue
				}
				ref, err := s.ImportFile(noteDir, noteName, p)
				if err != nil {
					cleanup()
					return "", err
				}
				return ref, nil
			}
		}
	}
	if s.Reader != nil {
		if ref, err := s.SaveFromClipboard(noteDir, noteName); err == nil {
			return ref, nil
		}
	}
	cleanup()
	return "", ErrNoImage
}

// isImageFile indica si p es un archivo regular con una extensión de imagen que el preview decodifica.
func isImageFile(p string) bool {
	if !imageExts[strings.ToLower(filepath.Ext(p))] {
		return false
	}
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// SaveFromClipboard guarda la imagen del portapapeles en <noteDir>/assets/ y
// devuelve la referencia relativa para markdown (siempre con "/").
func (s *Saver) SaveFromClipboard(noteDir, noteName string) (string, error) {
	assets, name, err := s.target(noteDir, noteName, ".png")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(assets, name)
	if err := s.Reader.ReadImage(dest); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	fi, err := os.Stat(dest)
	if err != nil || fi.Size() == 0 {
		_ = os.Remove(dest)
		return "", errors.New("no hay una imagen en el portapapeles")
	}
	return ref(name), nil
}

// ImportFile copia el archivo de imagen src a <noteDir>/assets/ (el original no
// se toca) y devuelve la referencia relativa para markdown.
func (s *Saver) ImportFile(noteDir, noteName, src string) (string, error) {
	assets, name, err := s.target(noteDir, noteName, strings.ToLower(filepath.Ext(src)))
	if err != nil {
		return "", err
	}
	dest := filepath.Join(assets, name)
	if err := copyFile(src, dest); err != nil {
		_ = os.Remove(dest)
		return "", err
	}
	return ref(name), nil
}

func ref(name string) string { return filepath.ToSlash(filepath.Join("assets", name)) }

// target prepara <noteDir>/assets y elige un nombre libre:
// <nota>-AAAAMMDD-HHMMSS[-n].<ext>.
func (s *Saver) target(noteDir, noteName, ext string) (assetsDir, name string, err error) {
	assetsDir = filepath.Join(noteDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return "", "", fmt.Errorf("no se pudo crear assets/: %w", err)
	}
	base := strings.TrimSuffix(strings.ToLower(noteName), ".md")
	base = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return -1
		}
		return r
	}, base)), "-")
	if base == "" {
		base = "imagen"
	}
	if s.now == nil {
		s.now = time.Now
	}
	stamp := s.now().Format("20060102-150405")
	name = fmt.Sprintf("%s-%s%s", base, stamp, ext)
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(assetsDir, name)); os.IsNotExist(err) {
			return assetsDir, name, nil
		}
		name = fmt.Sprintf("%s-%s-%d%s", base, stamp, n, ext)
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// imageExts son los formatos que el preview sabe decodificar.
var imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true}

// ParsePastedPaths interpreta el texto de un pegado como rutas de archivos de
// imagen que existen (una por línea; admite file://, comillas, "~/" y espacios
// escapados con "\"). Devuelve nil si el texto no es solo eso.
func ParsePastedPaths(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p, ok := pastedPath(line)
		if !ok {
			return nil
		}
		out = append(out, p)
	}
	return out
}

func pastedPath(s string) (string, bool) {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		s = s[1 : len(s)-1]
	}
	if rest, ok := strings.CutPrefix(s, "file://"); ok {
		u, err := url.Parse("file://" + rest)
		if err != nil {
			return "", false
		}
		s = filePathFromURL(u.Path, runtime.GOOS == "windows")
	} else if runtime.GOOS != "windows" {
		s = unescapeShell(s)
	}
	if rest, ok := strings.CutPrefix(s, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		s = filepath.Join(home, rest)
	}
	if !filepath.IsAbs(s) || !imageExts[strings.ToLower(filepath.Ext(s))] {
		return "", false
	}
	fi, err := os.Stat(s)
	if err != nil || !fi.Mode().IsRegular() {
		return "", false
	}
	return s, true
}

// filePathFromURL convierte el path de una URL file:// en una ruta del sistema.
// En Windows la URL es file:///C:/ruta: sobra la barra de antes de la unidad y
// las barras pasan a invertidas.
func filePathFromURL(p string, windows bool) string {
	if !windows {
		return p
	}
	if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return strings.ReplaceAll(p, "/", `\`)
}

// unescapeShell quita la barra de los caracteres escapados ("\ " -> " ").
func unescapeShell(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// SystemReader lee la imagen del portapapeles con las herramientas del
// sistema: osascript o pngpaste en macOS, wl-paste o xclip en Linux y
// PowerShell en Windows.
type SystemReader struct{}

// ReadImage escribe en dest la imagen del portapapeles como PNG.
func (SystemReader) ReadImage(dest string) error {
	switch runtime.GOOS {
	case "darwin":
		if path, err := exec.LookPath("pngpaste"); err == nil {
			return run(exec.Command(path, dest))
		}
		script := fmt.Sprintf(`
			set targetFile to (POSIX file %q)
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
		`, dest)
		return run(exec.Command("osascript", "-e", script))
	case "linux":
		if path, err := exec.LookPath("wl-paste"); err == nil {
			return runToFile(exec.Command(path, "--type", "image/png"), dest)
		}
		if path, err := exec.LookPath("xclip"); err == nil {
			return runToFile(exec.Command(path, "-selection", "clipboard", "-t", "image/png", "-o"), dest)
		}
		return errors.New("se requiere 'wl-paste' o 'xclip' para pegar imágenes en Linux")
	case "windows":
		ps := fmt.Sprintf(`
			Add-Type -AssemblyName System.Windows.Forms
			Add-Type -AssemblyName System.Drawing
			$img = [System.Windows.Forms.Clipboard]::GetImage()
			if ($img -ne $null) { $img.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png) } else { exit 1 }
		`, psQuote(dest))
		return run(exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps))
	}
	return fmt.Errorf("sistema operativo no soportado: %s", runtime.GOOS)
}

// ReadFiles devuelve las rutas de los archivos copiados en el portapapeles: osascript
// en macOS, wl-paste o xclip (text/uri-list) en Linux y PowerShell en Windows.
func (SystemReader) ReadFiles() ([]string, error) {
	var out []byte
	var err error
	switch runtime.GOOS {
	case "darwin":
		out, err = exec.Command("osascript", "-e", `POSIX path of (the clipboard as «class furl»)`).Output()
	case "linux":
		if path, lerr := exec.LookPath("wl-paste"); lerr == nil {
			out, err = exec.Command(path, "--type", "text/uri-list").Output()
		} else if path, lerr := exec.LookPath("xclip"); lerr == nil {
			out, err = exec.Command(path, "-selection", "clipboard", "-t", "text/uri-list", "-o").Output()
		} else {
			return nil, errors.New("se requiere 'wl-paste' o 'xclip' para leer archivos copiados en Linux")
		}
	case "windows":
		out, err = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			`Get-Clipboard -Format FileDropList | ForEach-Object { $_.FullName }`).Output()
	default:
		return nil, fmt.Errorf("sistema operativo no soportado: %s", runtime.GOOS)
	}
	if err != nil {
		return nil, fmt.Errorf("no hay archivos copiados (%v)", err)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if p, ok := pastedPath(line); ok {
			paths = append(paths, p)
		} else if filepath.IsAbs(line) && isImageFile(line) { // osascript y PowerShell imprimen la ruta tal cual
			paths = append(paths, line)
		}
	}
	return paths, nil
}

func run(cmd *exec.Cmd) error {
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("no hay una imagen en el portapapeles (%v: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func runToFile(cmd *exec.Cmd, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	cmd.Stdout = f
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no hay una imagen en el portapapeles (%v)", err)
	}
	return nil
}

// psQuote escapa una ruta para un literal entre comillas simples de PowerShell. PowerShell trata como comilla simple
// no solo ' sino también las tipográficas ‘ ’ ‚ ‛ (U+2018, U+2019, U+201A, U+201B): todas se duplican, si no una
// nota llamada `a’);calc;(’.md` cerraría el literal.
func psQuote(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\'', '\u2018', '\u2019', '\u201A', '\u201B':
			b.WriteRune(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
