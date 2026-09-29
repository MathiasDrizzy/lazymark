package i18n

import (
	"testing"
)

func TestI18n(t *testing.T) {
	SetLanguage("es")
	if CurrentLanguage() != LangES {
		t.Fatalf("se esperaba idioma 'es'")
	}
	if T("Hola", "Hello") != "Hola" {
		t.Errorf("traducción en español falló")
	}

	SetLanguage("en")
	if CurrentLanguage() != LangEN {
		t.Fatalf("se esperaba idioma 'en'")
	}
	if T("Hola", "Hello") != "Hello" {
		t.Errorf("traducción en inglés falló")
	}

	next := ToggleLanguage()
	if next != LangES {
		t.Errorf("ToggleLanguage debió cambiar a 'es'")
	}
}
