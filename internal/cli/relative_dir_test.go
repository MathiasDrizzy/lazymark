package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRelativeDirCLI(t *testing.T) {
	dir := fixture(t)
	content := "# Note\n#café #日本 #123\n```c\n#include\n```\n`#inline` https://x/#section\n- [ ] task\n"
	if err := os.WriteFile(filepath.Join(dir, "tags.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	id := taskID(t, dir, "task", "tags.md")
	link := filepath.Join(filepath.Dir(dir), "linked")
	if err := os.Symlink(dir, link); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{filepath.Base(dir), ".", "linked"} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(filepath.Dir(dir))
			if name == "." {
				t.Chdir(dir)
			}
			out, err := run(t, name, "note", "show", "tags.md")
			if err != nil || out != content {
				t.Fatalf("show: %v %q", err, out)
			}
			out, err = run(t, name, "note", "show", "tags.md", "--json")
			var shown struct{ ID string }
			if err != nil || json.Unmarshal([]byte(out), &shown) != nil || shown.ID != "tags.md" {
				t.Fatalf("show JSON: %v %s", err, out)
			}
			if got := taskID(t, name, "task", "tags.md"); got != id {
				t.Fatalf("id %q, want %q", got, id)
			}
			if out, err := run(t, name, "task", "toggle", id); err != nil {
				t.Fatalf("toggle: %v %s", err, out)
			}
			// Restore the fixture so each spelling exercises the same input.
			if _, err := run(t, name, "task", "toggle", id); err != nil {
				t.Fatal(err)
			}
			out, err = run(t, name, "note", "list", "--json")
			var notes []struct {
				ID   string
				Tags []string
			}
			if err != nil || json.Unmarshal([]byte(out), &notes) != nil {
				t.Fatalf("list: %v %s", err, out)
			}
			found := false
			for _, n := range notes {
				if n.ID == "tags.md" {
					found = true
					if !reflect.DeepEqual(n.Tags, []string{"café", "日本"}) {
						t.Fatalf("tags: %v", n.Tags)
					}
				}
			}
			if !found {
				t.Fatal("missing note")
			}
			out, err = run(t, name, "note", "new", "Created "+name, "--empty", "--json")
			var created struct{ ID string }
			if err != nil || json.Unmarshal([]byte(out), &created) != nil || filepath.IsAbs(created.ID) {
				t.Fatalf("new: %v %s", err, out)
			}
			if _, err := run(t, name, "note", "show", created.ID); err != nil {
				t.Fatal(err)
			}

		})
	}
}
