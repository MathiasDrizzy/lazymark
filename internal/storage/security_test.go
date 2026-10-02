package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks en Windows requieren privilegios")
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

// TestTrashMetaCannotDeleteOutside (S1, alta): un trash.json con ids o rutas hostiles (dentro de la carpeta de
// notas, o sea contenido no confiable) nunca borra ni mueve nada fuera de .trash/ y de la carpeta de notas.
func TestTrashMetaCannotDeleteOutside(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notas")
	victim := filepath.Join(root, "victima")
	os.MkdirAll(filepath.Join(notes, ".trash"), 0o755)
	os.MkdirAll(victim, 0o755)
	os.WriteFile(filepath.Join(victim, "importante.txt"), []byte("dato"), 0o644)
	os.WriteFile(filepath.Join(notes, "dentro.md"), []byte("# d\n"), 0o644)

	old := time.Now().AddDate(0, 0, -400)
	mk := func(id, orig string) TrashItem {
		return TrashItem{ID: id, Name: "x", OriginalPath: orig, DeletedAt: old, IsDir: true}
	}
	metaBytes, _ := json.Marshal([]TrashItem{ // json.Marshal: las rutas de Windows llevan "\\"
		mk("../../victima", "/nonexistent/x"),
		mk("../dentro.md", filepath.Join(notes, "y.md")),
		mk("/"+strings.TrimPrefix(filepath.ToSlash(victim), "/"), "x"),
		mk("..", "x"),
	})
	os.WriteFile(filepath.Join(notes, ".trash", "trash.json"), metaBytes, 0o644)

	s := New(notes)
	items, err := s.ListTrash() // lo que hace la TUI al abrir
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("las entradas inválidas no deben listarse: %+v", items)
	}
	if _, err := os.Stat(filepath.Join(victim, "importante.txt")); err != nil {
		t.Fatalf("ListTrash borró algo fuera de la papelera: %v", err)
	}
	if _, err := os.Stat(filepath.Join(notes, "dentro.md")); err != nil {
		t.Fatalf("ListTrash borró una nota: %v", err)
	}
	if len(s.TrashIssues()) != 4 {
		t.Errorf("las entradas ignoradas se reportan: %v", s.TrashIssues())
	}
	for _, id := range []string{"../../victima", "../dentro.md", "..", "/" + victim} {
		s.DeleteTrashItem(id)
	}
	if _, err := os.Stat(filepath.Join(victim, "importante.txt")); err != nil {
		t.Fatalf("DeleteTrashItem borró fuera: %v", err)
	}
}

// TestRestoreTrashConfinesDestination (S1): restaurar no coloca archivos fuera de la carpeta de notas, ni siquiera
// con un original_path hostil o un enlace simbólico en el camino.
func TestRestoreTrashConfinesDestination(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notas")
	outside := filepath.Join(root, "fuera")
	os.MkdirAll(filepath.Join(notes, ".trash"), 0o755)
	os.MkdirAll(outside, 0o755)
	os.WriteFile(filepath.Join(notes, ".trash", "1_a.md"), []byte("carga"), 0o644)
	os.WriteFile(filepath.Join(notes, ".trash", "2_b.md"), []byte("carga"), 0o644)
	symlinkOrSkip(t, outside, filepath.Join(notes, "enlace"))
	metaBytes, _ := json.Marshal([]TrashItem{
		{ID: "1_a.md", Name: "a.md", OriginalPath: filepath.Join(outside, "a.md"), DeletedAt: time.Now()},
		{ID: "2_b.md", Name: "b.md", OriginalPath: filepath.Join(notes, "enlace", "b.md"), DeletedAt: time.Now()},
	})
	os.WriteFile(filepath.Join(notes, ".trash", "trash.json"), metaBytes, 0o644)

	s := New(notes)
	for _, id := range []string{"1_a.md", "2_b.md"} {
		if err := s.RestoreTrashItem(id); err == nil {
			t.Errorf("restaurar %s debía fallar", id)
		}
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Errorf("se colocó algo fuera de la carpeta de notas: %v", entries)
	}
	if _, err := os.Stat(filepath.Join(notes, ".trash", "1_a.md")); err != nil {
		t.Error("el archivo de la papelera no debe perderse al fallar")
	}
	// una restauración legítima sigue funcionando
	p := filepath.Join(notes, "sub", "ok.md")
	os.MkdirAll(filepath.Dir(p), 0o755)
	os.WriteFile(p, []byte("# ok\n"), 0o644)
	item, err := s.MoveToTrash(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreTrashItem(item.ID); err != nil {
		t.Fatalf("restaurar lo propio: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Error("no volvió a su sitio")
	}
}

// TestRewriteIgnoresPredictableTmpSymlink (S2): un symlink <nota>.md.tmp no hace que una escritura caiga en su destino.
func TestRewriteIgnoresPredictableTmpSymlink(t *testing.T) {
	notes, note, outside := layout(t)
	symlinkOrSkip(t, outside, note+".tmp")
	before, _ := os.ReadFile(outside)
	s := New(notes)
	if err := s.MoveTask(note, 2, DefaultColumns, 2, time.Time{}); err != nil {
		t.Fatalf("mover: %v", err)
	}
	if after, _ := os.ReadFile(outside); string(after) != string(before) {
		t.Errorf("se sobrescribió el destino del symlink:\n%s", after)
	}
	if fi, _ := os.Lstat(note); fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
		t.Error("la nota debe seguir siendo un archivo regular")
	}
	if b, _ := os.ReadFile(note); !strings.Contains(string(b), "- [x] dentro") {
		t.Errorf("la nota no se actualizó: %s", b)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(note), ".lazymark-*")); len(left) != 0 {
		t.Errorf("quedaron temporales: %v", left)
	}
}

// TestCreateNoteDoesNotFollowSymlink (S3): una nota nueva sobre un symlink (colgante o no) no crea ni toca nada
// fuera, y el error es "ya existe".
func TestCreateNoteDoesNotFollowSymlink(t *testing.T) {
	notes, _, outside := layout(t)
	created := filepath.Join(filepath.Dir(outside), "creado.md")
	symlinkOrSkip(t, created, filepath.Join(notes, "nueva.md"))
	symlinkOrSkip(t, outside, filepath.Join(notes, "existente.md"))
	before, _ := os.ReadFile(outside)
	s := New(notes)
	for _, title := range []string{"nueva", "existente"} {
		if _, err := s.CreateNoteInDir(notes, title); !errors.Is(err, ErrNoteExists) {
			t.Errorf("%s: se esperaba ErrNoteExists: %v", title, err)
		}
	}
	if _, err := os.Stat(created); err == nil {
		t.Error("se creó un archivo fuera de la carpeta de notas")
	}
	if after, _ := os.ReadFile(outside); string(after) != string(before) {
		t.Error("se tocó el destino del symlink")
	}
	symlinkOrSkip(t, filepath.Join(filepath.Dir(outside), "dir-fuera"), filepath.Join(notes, "carpeta"))
	if _, err := s.CreateFolderInDir(notes, "carpeta"); err == nil {
		t.Error("una carpeta sobre un symlink debía fallar")
	}
}

// TestListNotesSkipsSymlinksOutside (S4): un .md que es un symlink a un archivo de fuera no aparece (ni sus tareas);
// uno que apunta dentro sí.
func TestListNotesSkipsSymlinksOutside(t *testing.T) {
	notes, note, outside := layout(t)
	os.WriteFile(outside, []byte("- [ ] API_KEY=SECRETO\n"), 0o644)
	symlinkOrSkip(t, outside, filepath.Join(notes, "enlace.md"))
	symlinkOrSkip(t, note, filepath.Join(notes, "interno.md"))
	s := New(notes)
	list, _ := s.ListNotes()
	var names []string
	for _, n := range list {
		names = append(names, n.ID)
		for _, tk := range n.Tasks {
			if strings.Contains(tk.Text, "SECRETO") {
				t.Errorf("tarea de un archivo de fuera: %+v", tk)
			}
		}
	}
	if strings.Contains(strings.Join(names, ","), "enlace.md") || !strings.Contains(strings.Join(names, ","), "interno.md") {
		t.Errorf("notas listadas: %v", names)
	}
	entries, _ := s.ListEntries()
	for _, e := range entries {
		if e.Name == "enlace.md" {
			t.Error("el árbol muestra el symlink de fuera")
		}
	}
}
