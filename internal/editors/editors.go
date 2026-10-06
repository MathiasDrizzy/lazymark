// Package editors instala y quita los plugins que traen micro, vim y nano para pegar
// imágenes desde el editor con `lazymark paste`. Cada instalación es idempotente, nunca
// pisa un archivo del usuario que no sea de lazymark y se deshace con Uninstall.
package editors

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"os"
	"path/filepath"
	"strings"
)

//go:embed files
var files embed.FS

// Names son los editores con plugin, en el orden en que se instalan.
var Names = []string{"micro", "vim", "nano"}

// Env es el entorno donde se instala: el HOME y cómo leer variables. Los tests lo
// apuntan a un directorio temporal; el HOME real solo se toca desde la línea de comandos.
type Env struct {
	Home   string
	Getenv func(string) string
}

// marker abre los archivos que son de lazymark; sin él, Install y Uninstall no los tocan.
const marker = "lazymark editor plugin (managed by `lazymark editor-plugins`"

func (e Env) get(k string) string {
	if e.Getenv == nil {
		return ""
	}
	return e.Getenv(k)
}

// configHome es $XDG_CONFIG_HOME o ~/.config.
func (e Env) configHome() string {
	if x := e.get("XDG_CONFIG_HOME"); x != "" {
		return x
	}
	return filepath.Join(e.Home, ".config")
}

// Install instala el plugin de editor y devuelve lo que hizo, una línea por acción.
func Install(editor string, e Env) ([]string, error) {
	switch editor {
	case "micro":
		return installMicro(e)
	case "vim":
		return installVim(e)
	case "nano":
		return installNano(e)
	}
	return nil, i18n.Errorf("editor desconocido %q (micro, vim o nano)", "unknown editor %q (micro, vim or nano)", editor)
}

// Uninstall deshace Install y devuelve lo que hizo. Si el plugin no está, no hace nada.
func Uninstall(editor string, e Env) ([]string, error) {
	switch editor {
	case "micro":
		return uninstallMicro(e)
	case "vim":
		return uninstallVim(e)
	case "nano":
		return uninstallNano(e)
	}
	return nil, i18n.Errorf("editor desconocido %q (micro, vim o nano)", "unknown editor %q (micro, vim or nano)", editor)
}

func asset(name string) string {
	b, err := files.ReadFile("files/" + name)
	if err != nil {
		panic(err) // los archivos van embebidos: un fallo es un error de compilación
	}
	return string(b)
}

// isOurs indica si el archivo existe y es de lazymark (su contenido lleva la marca).
func isOurs(path string) (exists, ours bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, false, nil
	}
	if err != nil {
		return true, false, err
	}
	return true, strings.Contains(string(b), marker), nil
}

// writeOwned escribe content en path si no existe o si es de lazymark; con un archivo
// ajeno falla sin tocarlo. Devuelve si cambió algo.
func writeOwned(path, content string) (changed bool, err error) {
	exists, ours, err := isOurs(path)
	if err != nil {
		return false, err
	}
	if exists && !ours {
		return false, i18n.Errorf("%s ya existe y no es de lazymark: no se toca", "%s already exists and is not lazymark's: left alone", path)
	}
	if exists {
		if b, _ := os.ReadFile(path); string(b) == content {
			return false, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(content), 0o644)
}

// removeOwned borra path si es de lazymark; con un archivo ajeno falla sin tocarlo.
func removeOwned(path string) (removed bool, err error) {
	exists, ours, err := isOurs(path)
	if err != nil || !exists {
		return false, err
	}
	if !ours {
		return false, i18n.Errorf("%s no es de lazymark: no se borra", "%s is not lazymark's: not deleted", path)
	}
	return true, os.Remove(path)
}

// pruneEmpty borra los directorios vacíos de dir hacia arriba hasta stop (sin incluirlo).
func pruneEmpty(dir, stop string) {
	for dir != stop && strings.HasPrefix(dir, stop) {
		if err := os.Remove(dir); err != nil { // falla si no está vacío: ahí se detiene
			return
		}
		dir = filepath.Dir(dir)
	}
}

// ─── micro ───────────────────────────────────────────────────────────────────
// Plugin Lua en <config>/plug/lazymark/ (https://github.com/zyedidia/micro/blob/master/runtime/help/plugins.md).
// La configuración de micro es $MICRO_CONFIG_HOME, o $XDG_CONFIG_HOME/micro, o ~/.config/micro
// (https://github.com/zyedidia/micro/blob/master/runtime/help/options.md).

func (e Env) microDir() string {
	if d := e.get("MICRO_CONFIG_HOME"); d != "" {
		return d
	}
	return filepath.Join(e.configHome(), "micro")
}

// microBindingWarning avisa si el usuario ya enlazó Alt-i en bindings.json: el plugin no la pisa
// (TryBindKey con overwrite=false), así que el atajo no funcionaría y hay que usar el comando.
func (e Env) microBindingWarning() []string {
	b, err := os.ReadFile(filepath.Join(e.microDir(), "bindings.json"))
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	for k, v := range m {
		if strings.EqualFold(k, "Alt-i") {
			return []string{fmt.Sprintf("micro: aviso: Alt-i ya está enlazada a %q en tu bindings.json y no se pisa; el atajo de lazymark no funcionará. Usa el comando `> pasteimage` (Ctrl-e) o cambia ese binding", fmt.Sprint(v))}
		}
	}
	return nil
}

func installMicro(e Env) ([]string, error) {
	done, err := installMicroFiles(e)
	if err != nil {
		return done, err
	}
	return append(done, e.microBindingWarning()...), nil
}

func installMicroFiles(e Env) ([]string, error) {
	dir := filepath.Join(e.microDir(), "plug", "lazymark")
	lua, repo := filepath.Join(dir, "lazymark.lua"), filepath.Join(dir, "repo.json")
	// un plug/lazymark/ con repo.json pero sin nuestro .lua es de otro plugin: no se toca
	if exists, _, _ := isOurs(lua); !exists {
		if _, err := os.Stat(repo); err == nil {
			return nil, i18n.Errorf("%s ya existe y no es de lazymark: no se toca", "%s already exists and is not lazymark's: left alone", repo)
		}
	}
	var done []string
	changed, err := writeOwned(lua, asset("micro/lazymark.lua"))
	if err != nil {
		return nil, err
	}
	if changed {
		done = append(done, "micro: "+lua)
	}
	// repo.json no lleva la marca, pero es nuestro: el .lua de su carpeta lo es
	want := asset("micro/repo.json")
	if b, err := os.ReadFile(repo); err != nil || string(b) != want {
		if err := os.WriteFile(repo, []byte(want), 0o644); err != nil {
			return done, err
		}
		done = append(done, "micro: "+repo)
	}
	return done, nil
}

func uninstallMicro(e Env) ([]string, error) {
	dir := filepath.Join(e.microDir(), "plug", "lazymark")
	lua := filepath.Join(dir, "lazymark.lua")
	exists, ours, err := isOurs(lua)
	if err != nil || !exists {
		return nil, err
	}
	if !ours {
		return nil, i18n.Errorf("%s no es de lazymark: no se borra", "%s is not lazymark's: not deleted", lua)
	}
	var done []string
	for _, f := range []string{"lazymark.lua", "repo.json"} {
		if err := os.Remove(filepath.Join(dir, f)); err == nil {
			done = append(done, "micro: borrado "+filepath.Join(dir, f))
		}
	}
	pruneEmpty(dir, e.microDir())
	return done, nil
}

// ─── vim ─────────────────────────────────────────────────────────────────────
// Paquete nativo de Vim: ~/.vim/pack/lazymark/start/lazymark/plugin/lazymark.vim
// (https://vimhelp.org/repeat.txt.html#packages).

func (e Env) vimFile() string {
	return filepath.Join(e.Home, ".vim", "pack", "lazymark", "start", "lazymark", "plugin", "lazymark.vim")
}

// vimMappingWarning avisa si algún vimrc del usuario ya mapea <Leader>ip: el plugin solo crea el
// mapeo si está libre, así que habría que usar :LazymarkPaste o remapear <Plug>(lazymark-paste).
func (e Env) vimMappingWarning() []string {
	for _, rc := range []string{
		filepath.Join(e.Home, ".vimrc"), filepath.Join(e.Home, ".vim", "vimrc"),
		filepath.Join(e.configHome(), "vim", "vimrc"), filepath.Join(e.Home, ".config", "vim", "vimrc"),
	} {
		b, err := os.ReadFile(rc)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			l := strings.ToLower(strings.ReplaceAll(line, " ", ""))
			if strings.Contains(l, "<leader>ip") && strings.Contains(l, "map") && !strings.HasPrefix(strings.TrimSpace(line), `"`) {
				return []string{fmt.Sprintf("vim: aviso: %s ya mapea <Leader>ip y el plugin no lo pisa. Usa `:LazymarkPaste` o mapea otra tecla: nmap <Leader>x <Plug>(lazymark-paste)", rc)}
			}
		}
	}
	return nil
}

func installVim(e Env) ([]string, error) {
	changed, err := writeOwned(e.vimFile(), asset("vim/lazymark.vim"))
	if err != nil {
		return nil, err
	}
	var done []string
	if changed {
		done = append(done, "vim: "+e.vimFile())
	}
	return append(done, e.vimMappingWarning()...), nil
}

func uninstallVim(e Env) ([]string, error) {
	removed, err := removeOwned(e.vimFile())
	if err != nil || !removed {
		return nil, err
	}
	pruneEmpty(filepath.Dir(e.vimFile()), filepath.Join(e.Home, ".vim"))
	return []string{"vim: borrado " + e.vimFile()}, nil
}

// ─── nano ────────────────────────────────────────────────────────────────────
// nano lee ~/.nanorc o $XDG_CONFIG_HOME/nano/nanorc o ~/.config/nano/nanorc
// (https://www.nano-editor.org/dist/latest/nanorc.5.html). `bind` no se permite en un archivo
// incluido ("Command "bind" not allowed in included file", verificado con GNU nano 7.2): el
// bloque va dentro del nanorc del usuario, entre dos líneas de marca que Uninstall reconoce.
// El resto del archivo no se toca. Si el nanorc ya enlaza M-7, Install se niega.

const (
	nanoBegin   = "# >>> lazymark (pegar imágenes; se quita con `lazymark editor-plugins uninstall nano`) >>>"
	nanoEnd     = "# <<< lazymark <<<"
	nanoCreated = "# lazymark: este archivo lo creó lazymark editor-plugins"
	nanoEOF     = " [eof]" // el nanorc del usuario no terminaba en salto de línea
)

// nanoRC elige el nanorc del usuario: el primero que exista de los tres sitios; si
// ninguno existe, ~/.nanorc.
func (e Env) nanoRC() string {
	for _, p := range []string{
		filepath.Join(e.Home, ".nanorc"),
		filepath.Join(e.configHome(), "nano", "nanorc"),
		filepath.Join(e.Home, ".config", "nano", "nanorc"),
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(e.Home, ".nanorc")
}

// nanoBlockRe y los helpers localizan el bloque de lazymark dentro de un nanorc.
func hasNanoBlock(text string) bool { return strings.Contains(text, "# >>> lazymark (") }

// bindsKey indica si text (sin el bloque de lazymark) enlaza la tecla key.
func bindsKey(text, key string) bool {
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "bind" && strings.EqualFold(f[1], key) {
			return true
		}
	}
	return false
}

func installNano(e Env) ([]string, error) {
	rc := e.nanoRC()
	cur, err := os.ReadFile(rc)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	text := string(cur)
	if hasNanoBlock(text) {
		return nil, nil // ya está
	}
	if bindsKey(text, "M-7") {
		return nil, i18n.Errorf("%s ya enlaza M-7 y no se pisa, así que no se instaló. Quita o cambia ese bind y vuelve a instalar, o agrega a mano con otra tecla libre: bind <tecla> \"{execute}lazymark paste --no-newline 2>/dev/null{enter}\" main", "%s already binds M-7 and it is not overridden, so nothing was installed. Remove or change that bind and install again, or add it by hand with another free key: bind <key> \"{execute}lazymark paste --no-newline 2>/dev/null{enter}\" main", rc)
	}
	block := asset("nano/lazymark.nanorc")
	add := block
	switch {
	case !exists:
		add = nanoCreated + "\n" + block
	case len(cur) > 0 && !strings.HasSuffix(text, "\n"):
		add = "\n" + strings.Replace(block, nanoBegin, nanoBegin+nanoEOF, 1)
	}
	if err := os.MkdirAll(filepath.Dir(rc), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(rc, append(cur, []byte(add)...), 0o644); err != nil {
		return nil, err
	}
	return []string{"nano: " + rc}, nil
}

func uninstallNano(e Env) ([]string, error) {
	rc := e.nanoRC()
	cur, err := os.ReadFile(rc)
	if err != nil || !hasNanoBlock(string(cur)) {
		return nil, nil
	}
	text := string(cur)
	block := asset("nano/lazymark.nanorc")
	switch eof, created := "\n"+strings.Replace(block, nanoBegin, nanoBegin+nanoEOF, 1), nanoCreated+"\n"+block; {
	case strings.Contains(text, eof):
		text = strings.Replace(text, eof, "", 1)
	case strings.Contains(text, created):
		text = strings.Replace(text, created, "", 1)
	case strings.Contains(text, block):
		text = strings.Replace(text, block, "", 1)
	default: // el usuario tocó el bloque: se quitan las líneas entre las dos marcas
		var keep []string
		inside, closed := false, false
		for _, line := range strings.SplitAfter(text, "\n") {
			switch {
			case strings.HasPrefix(line, "# >>> lazymark ("):
				inside = true
			case inside && strings.HasPrefix(line, nanoEnd):
				inside, closed = false, true
			case !inside:
				keep = append(keep, line)
			}
		}
		if !closed { // sin la línea de cierre no se sabe dónde acaba el bloque: borrar hasta el final se llevaría el resto del nanorc del usuario
			return nil, i18n.Errorf("%s: el bloque de lazymark no tiene su línea de cierre (%q): no se borra nada; quita el bloque a mano", "%s: lazymark's block has no closing line (%q): nothing is deleted; remove the block by hand", rc, nanoEnd)
		}
		text = strings.Join(keep, "")
	}
	if text == "" && strings.Contains(string(cur), nanoCreated) {
		return []string{"nano: borrado " + rc}, os.Remove(rc)
	}
	return []string{"nano: " + rc + " (sin el bloque de lazymark)"}, os.WriteFile(rc, []byte(text), 0o644)
}
