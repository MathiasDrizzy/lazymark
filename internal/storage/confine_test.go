package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

var timeZero time.Time

// layout crea una carpeta de notas con una nota y, fuera de ella, un archivo "secreto" con una línea de tarea.
func layout(t *testing.T) (notes, note, outside string) {
	t.Helper()
	root := t.TempDir()
	notes = filepath.Join(root, "notas")
	os.MkdirAll(filepath.Join(notes, "sub"), 0o755)
	note = filepath.Join(notes, "sub", "a.md")
	os.WriteFile(note, []byte("# A\n- [ ] dentro\n"), 0o644)
	outside = filepath.Join(root, "secreto.md")
	os.WriteFile(outside, []byte("clave\n- [ ] fuera\n"), 0o644)
	return
}

// TestResolveNoteConfinesToNotesDir: ResolveNote solo acepta archivos .md reales dentro de la carpeta de
// notas (rutas absolutas o relativas a ella); rechaza lo de fuera, "..", symlinks que salen y lo que no es .md.
func TestResolveNoteConfinesToNotesDir(t *testing.T) {
	notes, note, outside := layout(t)
	s := New(notes)

	for _, in := range []string{note, filepath.Join("sub", "a.md"), filepath.Join(notes, "sub", "..", "sub", "a.md")} {
		got, err := s.ResolveNote(in)
		if err != nil {
			t.Errorf("%q debía aceptarse: %v", in, err)
		}
		if want, _ := filepath.EvalSymlinks(note); got != want {
			t.Errorf("%q -> %q, se esperaba %q", in, got, want)
		}
	}

	bad := map[string]string{
		"absoluta fuera":      outside,
		"con ..":              filepath.Join(notes, "..", "secreto.md"),
		"relativa con ..":     filepath.Join("..", "secreto.md"),
		"no es .md":           filepath.Join(notes, "sub"),
		"no existe":           filepath.Join(notes, "nada.md"),
		"vacía":               "",
		"un directorio .md":   filepath.Join(notes, "d.md"),
		"fuera de otra forma": "/etc/hosts",
	}
	os.Mkdir(filepath.Join(notes, "d.md"), 0o755)
	for name, in := range bad {
		if _, err := s.ResolveNote(in); !errors.Is(err, ErrOutsideNotes) && err == nil {
			t.Errorf("%s: %q debía rechazarse", name, in)
		}
	}
	if _, err := s.ResolveNote(outside); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("una ruta fuera debe dar ErrOutsideNotes: %v", err)
	}

	if runtime.GOOS != "windows" {
		// un symlink dentro de la carpeta que apunta a un archivo de fuera
		os.Symlink(outside, filepath.Join(notes, "enlace.md"))
		if _, err := s.ResolveNote(filepath.Join(notes, "enlace.md")); !errors.Is(err, ErrOutsideNotes) {
			t.Errorf("un symlink que sale de la carpeta debe rechazarse: %v", err)
		}
		// un symlink a una carpeta de fuera
		outDir := filepath.Join(filepath.Dir(notes), "fuera")
		os.MkdirAll(outDir, 0o755)
		os.WriteFile(filepath.Join(outDir, "b.md"), []byte("x"), 0o644)
		os.Symlink(outDir, filepath.Join(notes, "carpeta"))
		if _, err := s.ResolveNote(filepath.Join(notes, "carpeta", "b.md")); !errors.Is(err, ErrOutsideNotes) {
			t.Errorf("un symlink de carpeta que sale debe rechazarse: %v", err)
		}
		// un symlink que se queda dentro sí vale
		os.Symlink(note, filepath.Join(notes, "alias.md"))
		if _, err := s.ResolveNote(filepath.Join(notes, "alias.md")); err != nil {
			t.Errorf("un symlink que apunta dentro debe aceptarse: %v", err)
		}
		// la carpeta de notas misma puede ser un symlink
		link := filepath.Join(filepath.Dir(notes), "enlace-notas")
		os.Symlink(notes, link)
		if _, err := New(link).ResolveNote(filepath.Join(link, "sub", "a.md")); err != nil {
			t.Errorf("con la carpeta de notas como symlink, sus notas deben aceptarse: %v", err)
		}
	}
}

// TestWritersRefuseOutsideNotes: las funciones que escriben una línea no tocan archivos de fuera de la carpeta de notas.
func TestWritersRefuseOutsideNotes(t *testing.T) {
	notes, note, outside := layout(t)
	s := New(notes)
	before, _ := os.ReadFile(outside)
	if _, err := s.ToggleTask(outside, 2); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("ToggleTask fuera: %v", err)
	}
	if err := s.UpdateTaskStage(outside, 2, StageDoing); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("UpdateTaskStage fuera: %v", err)
	}
	if err := s.AppendToNote(outside, "x", timeZero); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("AppendToNote fuera: %v", err)
	}
	if err := s.InsertAfterLine(outside, 1, "x", timeZero); !errors.Is(err, ErrOutsideNotes) {
		t.Errorf("InsertAfterLine fuera: %v", err)
	}
	if after, _ := os.ReadFile(outside); string(after) != string(before) {
		t.Errorf("se modificó un archivo de fuera: %q", after)
	}
	// y dentro sigue funcionando
	if done, err := s.ToggleTask(note, 2); err != nil || !done {
		t.Errorf("ToggleTask dentro: %v %v", done, err)
	}
}
