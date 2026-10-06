package ops

import (
	"path/filepath"
	"strings"
	"testing"

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
