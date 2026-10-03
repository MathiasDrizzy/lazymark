// Package i18n traduce los textos de la interfaz. El código escribe cada texto como un par español/inglés,
// i18n.T("Notas", "Notes"); el español y el inglés salen de ese par y los demás idiomas (pt, fr, de, it, ja, zh) de los
// catálogos de catalog/<idioma>.json, que tienen como clave el texto en inglés. Un texto sin traducción cae en
// inglés; el test de cobertura (keys_test.go, catalog_test.go) impide que quede alguno sin traducir.
package i18n

import (
	"embed"
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Language es un idioma de la interfaz, por su código de dos letras.
type Language string

const (
	LangEN Language = "en"
	LangES Language = "es"
	LangPT Language = "pt" // portugués de Brasil (pt-BR)
	LangFR Language = "fr"
	LangDE Language = "de"
	LangIT Language = "it"
	LangJA Language = "ja"
	LangZH Language = "zh" // chino simplificado (zh-CN)
)

// Languages son los idiomas soportados, en el orden en que los recorre Ajustes.
var Languages = []Language{LangEN, LangES, LangPT, LangFR, LangDE, LangIT, LangJA, LangZH}

// Name es el nombre del idioma en ese mismo idioma (el que se muestra en Ajustes).
func (l Language) Name() string {
	switch l {
	case LangES:
		return "Español"
	case LangPT:
		return "Português (BR)"
	case LangFR:
		return "Français"
	case LangDE:
		return "Deutsch"
	case LangIT:
		return "Italiano"
	case LangJA:
		return "日本語"
	case LangZH:
		return "简体中文"
	}
	return "English"
}

//go:embed catalog/*.json
var catalogFS embed.FS

var (
	catalogOnce sync.Once
	catalogs    map[Language]map[string]string
)

// catalog devuelve las traducciones de l (clave: texto en inglés), leídas una vez.
func catalog(l Language) map[string]string {
	catalogOnce.Do(func() {
		catalogs = map[Language]map[string]string{}
		for _, lang := range Languages {
			data, err := catalogFS.ReadFile("catalog/" + string(lang) + ".json")
			if err != nil {
				continue // en y es no tienen catálogo
			}
			m := map[string]string{}
			if err := json.Unmarshal(data, &m); err != nil {
				panic("lazymark: el catálogo de " + string(lang) + " está mal formado: " + err.Error()) // un error del repo
			}
			catalogs[lang] = m
		}
	})
	return catalogs[l]
}

var (
	mu          sync.RWMutex
	currentLang = Detect(os.Getenv)
)

// Parse reconoce un idioma por su código (en, es, pt, pt-BR, pt_BR, zh-CN…) o por su nombre.
func Parse(s string) (Language, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, ".@"); i >= 0 { // "pt_BR.UTF-8"
		s = s[:i]
	}
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	switch s {
	case "en", "english", "ingles", "inglés":
		return LangEN, true
	case "es", "spanish", "español", "espanol":
		return LangES, true
	case "pt", "portuguese", "português", "portugues":
		return LangPT, true
	case "fr", "french", "français", "francais":
		return LangFR, true
	case "de", "german", "deutsch":
		return LangDE, true
	case "it", "italian", "italiano":
		return LangIT, true
	case "ja", "jp", "japanese", "日本語":
		return LangJA, true
	case "zh", "chinese", "中文", "简体中文":
		return LangZH, true
	}
	return LangEN, false
}

// Detect elige el idioma del sistema con las variables de entorno de locale, en el orden de POSIX (LC_ALL, LC_MESSAGES,
// LANG): la primera que esté puesta decide ("pt_BR.UTF-8" → pt). Un idioma que no se soporta, o ninguna variable: inglés.
func Detect(getenv func(string) string) Language {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(getenv(name))
		if v == "" || v == "C" || v == "POSIX" {
			continue
		}
		if l, ok := Parse(v); ok {
			return l
		}
		return LangEN
	}
	return LangEN
}

// SetLanguage define el idioma actual: un código o nombre soportado, o "auto" (el del sistema). Otro valor: inglés.
func SetLanguage(lang string) {
	l := LangEN
	if s := strings.ToLower(strings.TrimSpace(lang)); s == "auto" || s == "" {
		l = Detect(os.Getenv)
	} else if p, ok := Parse(s); ok {
		l = p
	}
	mu.Lock()
	currentLang = l
	mu.Unlock()
}

// CurrentLanguage devuelve el idioma activo.
func CurrentLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// ToggleLanguage pasa al siguiente idioma de la lista y lo devuelve.
func ToggleLanguage() Language {
	cur := CurrentLanguage()
	next := Languages[0]
	for i, l := range Languages {
		if l == cur {
			next = Languages[(i+1)%len(Languages)]
		}
	}
	mu.Lock()
	currentLang = next
	mu.Unlock()
	return next
}

// T devuelve el texto en el idioma activo: es y en vienen del código; los demás idiomas, del catálogo (clave: en), y sin
// traducción cae en inglés.
func T(es, en string) string {
	switch l := CurrentLanguage(); l {
	case LangES:
		return es
	case LangEN:
		return en
	default:
		if s := catalog(l)[en]; s != "" {
			return s
		}
		return en
	}
}
