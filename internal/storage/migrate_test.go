package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const migrateFixture = "# Migración\r\n" +
	"\r\n" +
	"Un párrafo con 📅 2026-05-10 que no es una tarea.\r\n" +
	"- [ ] emoji 🛫 2026-05-01 📅 2026-05-10\r\n" +
	"- [x] hecha ✅ 2026-05-09 ⏳ 2026-05-02 ➕ 2026-04-30\r\n" +
	"  - [ ] anidada [due:: 2026-06-01] (start:: 2026-05-20)\r\n" +
	"1. [ ] numerada 📅️ 2026-07-01 #kb/doing\r\n" +
	"- [ ] inválida 📅 2026-02-30 y [due:: 2026-13-01]\r\n" +
	"```\r\n" +
	"- [ ] en un bloque de código 📅 2026-05-10\r\n" +
	"```\r\n" +
	"- [ ] sin fechas\r\n"

// TestMigrateDates (ORD-017 F3): pasa las fechas de las tareas de un formato al otro: solo toca líneas de tarea (no párrafos ni código), conserva el resto de la
// línea y el CRLF, no toca las fechas inválidas, y es idempotente. Con dry-run devuelve el diff y no escribe nada.
func TestMigrateDates(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.md")
	os.WriteFile(p, []byte(migrateFixture), 0o644)
	os.WriteFile(filepath.Join(dir, "otra.md"), []byte("- [ ] ya dataview [due:: 2026-05-10]\n"), 0o644)
	s := New(dir)

	// dry-run a dataview: diff correcto, nada escrito
	diffs, err := s.MigrateDates(FormatDataview, true)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != migrateFixture {
		t.Fatal("dry-run no debe escribir")
	}
	got := map[int]string{}
	for _, d := range diffs {
		if filepath.Base(d.Path) == "m.md" {
			got[d.Line] = d.After
		}
	}
	want := map[int]string{
		4: "- [ ] emoji [start:: 2026-05-01] [due:: 2026-05-10]",
		5: "- [x] hecha [completion:: 2026-05-09] [scheduled:: 2026-05-02] [created:: 2026-04-30]",
		6: "  - [ ] anidada [due:: 2026-06-01] [start:: 2026-05-20]",
		7: "1. [ ] numerada [due:: 2026-07-01] #kb/doing",
	}
	if len(got) != len(want) {
		t.Errorf("líneas a cambiar: %v, se esperaban %v", got, want)
	}
	for n, w := range want {
		if got[n] != w {
			t.Errorf("línea %d: %q, se esperaba %q", n, got[n], w)
		}
	}
	// migrar de verdad
	if _, err := s.MigrateDates(FormatDataview, false); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	lines := strings.Split(string(b), "\r\n")
	orig := strings.Split(migrateFixture, "\r\n")
	for i := range orig {
		if w, ok := want[i+1]; ok {
			if lines[i] != w {
				t.Errorf("archivo, línea %d: %q", i+1, lines[i])
			}
		} else if lines[i] != orig[i] {
			t.Errorf("la línea %d no debía cambiar: %q → %q", i+1, orig[i], lines[i])
		}
	}
	if !strings.Contains(string(b), "\r\n") || strings.Contains(strings.ReplaceAll(string(b), "\r\n", ""), "\n") {
		t.Error("el CRLF se conserva")
	}
	// idempotente
	again, err := s.MigrateDates(FormatDataview, false)
	if err != nil || len(again) != 0 {
		t.Errorf("la segunda corrida no cambia nada: %v %d líneas", err, len(again))
	}
	if b2, _ := os.ReadFile(p); string(b2) != string(b) {
		t.Error("la segunda corrida no debe reescribir el archivo")
	}
	// de vuelta a emojis: las mismas líneas, y de nuevo idempotente
	back, err := s.MigrateDates(FormatEmoji, false)
	if err != nil || len(back) < 4 {
		t.Fatalf("de vuelta a emojis: %v %d", err, len(back))
	}
	b3, _ := os.ReadFile(p)
	if !strings.Contains(string(b3), "- [ ] emoji 🛫 2026-05-01 📅 2026-05-10\r\n") || !strings.Contains(string(b3), "  - [ ] anidada 📅 2026-06-01 🛫 2026-05-20\r\n") {
		t.Errorf("de vuelta:\n%s", b3)
	}
	if n, _ := s.MigrateDates(FormatEmoji, false); len(n) != 0 {
		t.Error("idempotente también hacia emojis")
	}
}

// TestMigrateStopsMidwayAndReports (ORD-017 rev 2, R2-3): si una nota cambia afuera mientras se migra, esa no se escribe, el error nombra la nota, lo ya migrado se
// devuelve y no se toca ninguna de las siguientes.
func TestMigrateStopsMidwayAndReports(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.md", "b.md", "c.md"} {
		os.WriteFile(filepath.Join(dir, n), []byte("- [ ] t 📅 2026-05-10\n"), 0o644)
	}
	migrateBeforeWrite = func(path string) {
		if filepath.Base(path) == "b.md" { // otro programa guarda b.md justo antes
			os.WriteFile(path, []byte("- [ ] t 📅 2026-05-10\notra línea\n"), 0o644)
			future := time.Now().Add(time.Hour)
			os.Chtimes(path, future, future)
		}
	}
	defer func() { migrateBeforeWrite = nil }()
	done, err := New(dir).MigrateDates(FormatDataview, false)
	if err == nil || !errors.Is(err, ErrNoteChanged) || !strings.Contains(err.Error(), "b.md") {
		t.Fatalf("el error nombra la nota que cambió: %v", err)
	}
	var names []string
	for _, c := range done {
		names = append(names, filepath.Base(c.Path))
	}
	if len(names) != 1 || names[0] != "a.md" {
		t.Errorf("solo a.md se migró antes del corte: %v", names)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.md")); string(b) != "- [ ] t [due:: 2026-05-10]\n" {
		t.Errorf("a.md: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "b.md")); string(b) != "- [ ] t 📅 2026-05-10\notra línea\n" {
		t.Errorf("b.md conserva el cambio de afuera: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "c.md")); string(b) != "- [ ] t 📅 2026-05-10\n" {
		t.Errorf("c.md no se toca tras el corte: %q", b)
	}
}
