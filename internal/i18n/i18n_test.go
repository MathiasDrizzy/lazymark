package i18n

import (
	"errors"
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
}

// TestErrorTextFollowsLanguage (ORD-019 C.7): un error centinela se escribe en el idioma de la interfaz EN EL MOMENTO de mostrarlo, aunque se haya creado antes de
// elegir el idioma; Errorf envuelve con %w y errors.Is sigue funcionando; un idioma sin catálogo cae en inglés, nunca en español.
func TestErrorTextFollowsLanguage(t *testing.T) {
	defer SetLanguage("es")
	base := NewError("no existe esa tarea", "no such task")
	wrapped := Errorf("%w: el id %q", "%w: the id %q", base, "x")
	SetLanguage("es")
	if got := base.Error(); got != "no existe esa tarea" {
		t.Errorf("es: %q", got)
	}
	SetLanguage("en")
	if got := base.Error(); got != "no such task" {
		t.Errorf("en: %q", got)
	}
	SetLanguage("fr")
	if got := base.Error(); got != "no such task" {
		t.Errorf("fr sin traducción cae en inglés: %q", got)
	}
	if !errors.Is(wrapped, base) || errors.Is(wrapped, NewError("no existe esa tarea", "no such task")) {
		t.Error("errors.Is compara la identidad del centinela")
	}
	SetLanguage("en")
	if got := Errorf("%w: el id %q", "%w: the id %q", base, "x").Error(); got != `no such task: the id "x"` {
		t.Errorf("Errorf en inglés: %q", got)
	}
	SetLanguage("es")
	if got := Errorf("%w: el id %q", "%w: the id %q", base, "x").Error(); got != `no existe esa tarea: el id "x"` {
		t.Errorf("Errorf en español: %q", got)
	}
}
