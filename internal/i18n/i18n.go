package i18n

import (
	"os"
	"strings"
)

type Language string

const (
	LangES Language = "es"
	LangEN Language = "en"
)

var currentLang = LangES

func init() {
	// Detectar idioma del sistema
	langEnv := strings.ToLower(os.Getenv("LANG") + " " + os.Getenv("LC_ALL") + " " + os.Getenv("LC_MESSAGES"))
	if strings.Contains(langEnv, "es") {
		currentLang = LangES
	} else {
		currentLang = LangEN
	}
}

// SetLanguage define el idioma actual ("es" o "en")
func SetLanguage(lang string) {
	switch strings.ToLower(lang) {
	case "es", "spanish", "español":
		currentLang = LangES
	case "en", "english", "ingles", "inglés":
		currentLang = LangEN
	default:
		currentLang = LangEN
	}
}

// CurrentLanguage devuelve el idioma activo
func CurrentLanguage() Language {
	return currentLang
}

// ToggleLanguage alterna entre español e inglés
func ToggleLanguage() Language {
	if currentLang == LangES {
		currentLang = LangEN
	} else {
		currentLang = LangES
	}
	return currentLang
}

// T devuelve la traducción según el idioma activo
func T(es, en string) string {
	if currentLang == LangES {
		return es
	}
	return en
}
