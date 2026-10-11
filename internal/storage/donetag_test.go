package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHasTag(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"ship it #archived", true},
		{"#archived ship it", true},
		{"ship it #ARCHIVED", true},
		{"ship it #archived-later", false},
		{"ship it #archived/sub", false},
		{"ship it#archived", false},
		{"ship it archived", false},
		{"ship it #kb/archived", false},
	}
	for _, c := range cases {
		if got := HasTag(c.text, "archived"); got != c.want {
			t.Errorf("HasTag(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestRewriteToggleTag(t *testing.T) {
	cases := []struct {
		line, want string
		added      bool
	}{
		{"- [x] ship it", "- [x] ship it #archived", true},
		{"- [x] ship it #archived", "- [x] ship it", false},
		{"- [x] ship it #ARCHIVED 📅 2026-10-12", "- [x] ship it 📅 2026-10-12", false},
		{"- [x] ship it ✅ 2026-10-11", "- [x] ship it ✅ 2026-10-11 #archived", true},
		{"- [ ] open  \r", "- [ ] open #archived\r", true},
		{"- [x] a #archived #kb/doing", "- [x] a #kb/doing", false},
	}
	for _, c := range cases {
		got, added := RewriteToggleTag(c.line, "archived")
		if got != c.want || added != c.added {
			t.Errorf("RewriteToggleTag(%q) = %q, %v; want %q, %v", c.line, got, added, c.want, c.added)
		}
	}
}

func TestToggleTaskTagTouchesOnlyThatLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.md")
	body := "# n\n\n- [x] one\n- [ ] two\ntext\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	added, err := s.ToggleTaskTag(path, 3, "archived", fileTime(t, path))
	if err != nil || !added {
		t.Fatalf("added=%v err=%v", added, err)
	}
	got, _ := os.ReadFile(path)
	if want := "# n\n\n- [x] one #archived\n- [ ] two\ntext\n"; string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	added, err = s.ToggleTaskTag(path, 3, "archived", fileTime(t, path))
	if err != nil || added {
		t.Fatalf("second toggle: added=%v err=%v", added, err)
	}
	if got, _ := os.ReadFile(path); string(got) != body {
		t.Fatalf("toggling twice must restore the note, got %q", got)
	}
}

func TestToggleTaskTagRefusesAChangedNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "n.md")
	if err := os.WriteFile(path, []byte("- [x] one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(dir)
	stale := fileTime(t, path).Add(-1e9)
	if _, err := s.ToggleTaskTag(path, 1, "archived", stale); err != ErrNoteChanged {
		t.Fatalf("got %v, want ErrNoteChanged", err)
	}
}

func fileTime(t *testing.T, path string) time.Time {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime()
}
