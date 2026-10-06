package ops

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// TestNoteIDOfDotDotNames (ORD-018, segunda opinión): una nota o carpeta cuyo nombre empieza con ".." dentro del vault sigue teniendo como id su ruta relativa.
func TestNoteIDOfDotDotNames(t *testing.T) {
	dir := t.TempDir()
	s := &Service{Store: storage.New(dir)}
	if got := s.noteID(filepath.Join(dir, "..oculto.md")); got != "..oculto.md" {
		t.Errorf("id = %q", got)
	}
	if got := s.noteID(filepath.Join(dir, "..d", "n.md")); got != "..d/n.md" {
		t.Errorf("id = %q", got)
	}
	if got := s.noteID(filepath.Join(filepath.Dir(dir), "fuera.md")); strings.HasPrefix(got, "..") {
		t.Errorf("fuera de la carpeta no se hace pasar por relativa: %q", got)
	}
}

// TestEmptyNoteNameIsUsageInEveryLanguage (ORD-019 rev 2 / L17): un título que no deja nombre (`///`) es un error de uso (código 2) igual en español que en inglés:
// el código no decide por el texto traducido del error, sino por un error centinela.
func TestEmptyNoteNameIsUsageInEveryLanguage(t *testing.T) {
	dir := t.TempDir()
	s := &Service{Store: storage.New(dir)}
	defer i18n.SetLanguage("es")
	for _, lang := range []string{"es", "en", "fr"} {
		i18n.SetLanguage(lang)
		for _, title := range []string{"///", "...", "   /   "} {
			_, err := s.NewNote(title, "", true)
			if err == nil {
				t.Fatalf("%s: %q no deja nombre: debe fallar", lang, title)
			}
			if code := Code(err); code != ExitUsage {
				t.Errorf("%s: %q: código %d (%v), se esperaba %d", lang, title, code, err, ExitUsage)
			}
		}
	}
}
