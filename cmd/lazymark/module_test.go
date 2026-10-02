package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/mod/module"
)

// TestTrackedFileNamesAreValidForModules: `go install …/cmd/lazymark@<versión>`
// funciona solo si Go puede crear el zip del módulo, y para eso todos los archivos
// versionados deben tener nombres válidos: Go prohíbe emoji y otros símbolos (un
// fixture con ⚠️ en el nombre rompió la instalación de cualquiera). Se revisan los
// archivos del índice de git con la misma función que usa Go para crear el zip, así
// el pre-commit atrapa el problema antes de que llegue a un commit.
func TestTrackedFileNamesAreValidForModules(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("sin git")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Skipf("no es un repositorio git: %v", err)
	}
	checked := 0
	for _, path := range strings.Split(string(out), "\x00") {
		if path == "" {
			continue
		}
		checked++
		if err := module.CheckFilePath(path); err != nil {
			t.Errorf("el archivo versionado %q no es válido en un módulo de Go (go install fallaría): %v", path, err)
		}
	}
	if checked == 0 {
		t.Skip("el repositorio no tiene archivos versionados")
	}
}
