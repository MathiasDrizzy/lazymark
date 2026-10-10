package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRelativeVaultOperations(t *testing.T) {
	parent := t.TempDir()
	vault := filepath.Join(parent, "vault")
	if err := os.Mkdir(vault, 0755); err != nil {
		t.Fatal(err)
	}
	absolute := New(vault)
	note, err := absolute.CreateNoteInDirWithBody("", "Test", "- [ ] task #real\n")
	if err != nil {
		t.Fatal(err)
	}
	ids := absolute.TaskIDs(*note)
	link := filepath.Join(parent, "link")
	if err := os.Symlink(vault, link); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vault", ".", "link"} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(parent)
			if name == "." {
				t.Chdir(vault)
			}
			store := New(name)
			if !filepath.IsAbs(store.BaseDir) {
				t.Fatal("BaseDir is relative")
			}
			if name == "link" && store.BaseDir != link {
				t.Fatal("symlink spelling lost")
			}
			if _, err := store.ResolveNote("test.md"); err != nil {
				t.Fatal(err)
			}
			n, task, err := store.FindTask(ids[0])
			if err != nil {
				t.Fatal(err)
			}
			if got := store.TaskIDs(n); !reflect.DeepEqual(got, ids) {
				t.Fatalf("IDs %v, want %v", got, ids)
			}
			if _, err := store.ToggleTask(task.NotePath, task.Line); err != nil {
				t.Fatal(err)
			}
			folder, err := store.ResolveFolder("")
			if err != nil {
				t.Fatal(err)
			}
			created, err := store.CreateNoteInDirWithBody(folder, "Created "+name, "- [ ] created #real\n")
			if err != nil {
				t.Fatal(err)
			}
			if got := store.TaskIDs(*created); !reflect.DeepEqual(got, absolute.TaskIDs(*created)) {
				t.Fatalf("created task IDs = %v, want %v", got, absolute.TaskIDs(*created))
			}
			if err := store.confineNewPath(created.Path); err != nil {
				t.Fatal(err)
			}
			if len(store.NotePaths()) == 0 {
				t.Fatal("no note paths")
			}
		})
	}
	t.Chdir(parent)
	if _, err := (&Storage{BaseDir: "vault"}).ResolveNote("test.md"); err != nil {
		t.Fatal(err)
	}
}
