package theme

import (
	"encoding/json"
	"errors"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/safeio"
)

// Temas del usuario: un archivo JSON por tema en la carpeta themes, junto a config.json. El nombre del tema es el
// del archivo sin ".json". Lleva los 14 colores de Palette con sus nombres en minúsculas, todos como "#rrggbb", y
// "source" si se quiere. Salen en la lista después de los incluidos, en orden alfabético.

// maxThemeBytes: un tema son 15 líneas; nada razonable pasa de esto.
const maxThemeBytes = 64 << 10

// UserThemeProblem es un tema que no se cargó. Va en español y en inglés con sus argumentos, como los avisos de
// config.json, para traducirlo cuando se muestra.
type UserThemeProblem struct {
	ES, EN string
	Args   []any
}

// userNames son los temas del usuario cargados, en el orden de la lista.
var userNames []string

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type paletteSlot struct {
	key string
	dst *color.Color
}

// paletteSlots relaciona cada clave del archivo con su color en p.
func paletteSlots(p *Palette) []paletteSlot {
	return []paletteSlot{
		{"base", &p.Base}, {"mantle", &p.Mantle}, {"surface0", &p.Surface0}, {"surface1", &p.Surface1},
		{"overlay0", &p.Overlay0}, {"text", &p.Text}, {"subtext0", &p.Subtext0}, {"peach", &p.Peach},
		{"mauve", &p.Mauve}, {"teal", &p.Teal}, {"green", &p.Green}, {"red", &p.Red},
		{"blue", &p.Blue}, {"yellow", &p.Yellow},
	}
}

// LoadUserThemes carga los temas de dir en lugar de los de la carga anterior. Si la carpeta no existe no pasa
// nada; un archivo que no vale se salta con un aviso y los demás se cargan igual.
func LoadUserThemes(dir string) []UserThemeProblem {
	for _, n := range userNames {
		delete(AvailableThemes, n)
	}
	userNames = nil

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return []UserThemeProblem{{"no se pudo leer la carpeta de temas %s: %v", "could not read the themes folder %s: %v", []any{dir, err}}}
	}
	var problems []UserThemeProblem
	for _, e := range entries { // ReadDir los da ordenados por nombre
		name, isTheme := strings.CutSuffix(e.Name(), ".json")
		if !isTheme || name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		p, problem := readUserTheme(filepath.Join(dir, e.Name()), name)
		if problem != nil {
			problems = append(problems, *problem)
			continue
		}
		AvailableThemes[name] = p
		userNames = append(userNames, name)
	}
	return problems
}

// readUserTheme lee un archivo de tema o dice por qué no vale.
func readUserTheme(path, name string) (Palette, *UserThemeProblem) {
	fail := func(es, en string, args ...any) (Palette, *UserThemeProblem) {
		return Palette{}, &UserThemeProblem{"el tema %s no se cargó: " + es, "theme %s was not loaded: " + en, append([]any{name}, args...)}
	}
	if _, builtin := AvailableThemes[name]; builtin {
		return fail("ya hay un tema incluido con ese nombre", "a built-in theme has that name")
	}
	data, err := safeio.ReadRegular(path, maxThemeBytes) // un FIFO llamado x.json no cuelga el arranque
	if err != nil {
		var pe *fs.PathError
		switch {
		case errors.Is(err, safeio.ErrNotRegular):
			err = safeio.ErrNotRegular // sin la ruta, que ya se sabe
		case errors.As(err, &pe):
			err = pe.Err
		}
		return fail("%v", "%v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil || raw == nil {
		return fail("no es un objeto JSON válido", "it is not a valid JSON object")
	}

	p := Palette{Name: name}
	slots := paletteSlots(&p)
	known := map[string]bool{"source": true}
	for _, slot := range slots {
		known[slot.key] = true
	}
	var unknown []string
	for k := range raw {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 { // se mira antes que los colores que faltan, así una errata como "bse" sale con su nombre
		sort.Strings(unknown)
		return fail("clave desconocida %q", "unknown key %q", unknown[0])
	}
	for _, slot := range slots {
		v, ok := raw[slot.key]
		if !ok {
			return fail("falta el color %q", "the color %q is missing", slot.key)
		}
		var s string
		if json.Unmarshal(v, &s) != nil || !hexColor.MatchString(s) {
			return fail("el color %q vale %s y no es #rrggbb", "the color %q is %s, not #rrggbb", slot.key, string(v))
		}
		*slot.dst = lipgloss.Color(s)
	}
	if v, ok := raw["source"]; ok && json.Unmarshal(v, &p.Source) != nil {
		return fail("\"source\" no es texto", "\"source\" is not a string")
	}
	return p, nil
}
