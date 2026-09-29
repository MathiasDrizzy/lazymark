package clipboard

import (
	"os"
	"testing"
)

func TestClipboardCreation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lazymark-clip-*")
	if err != nil {
		t.Fatalf("Fallo temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	s := New(tempDir)
	if s.AssetsDir != tempDir {
		t.Errorf("Directorio de assets inválido: esperado %s, obtenido %s", tempDir, s.AssetsDir)
	}
}
