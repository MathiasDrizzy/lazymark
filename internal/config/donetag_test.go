package config

import "testing"

func TestDoneTagDefaultsKeepTheOldBehaviour(t *testing.T) {
	d := DefaultConfig("")
	if d.HideCompletedKanban {
		t.Error("hide_completed_kanban must default to false: upgraders keep seeing finished cards")
	}
	if d.DoneTag != "lm-done" || d.DoneTagHides != "both" {
		t.Errorf("done_tag = %q, done_tag_hides = %q; want lm-done and both", d.DoneTag, d.DoneTagHides)
	}
	if !d.DoneTagHidesKanban() || !d.DoneTagHidesTasks() {
		t.Error("with the default scope the tag hides in both views")
	}
}

func TestDoneTagHidesScope(t *testing.T) {
	for scope, want := range map[string][2]bool{
		"both": {true, true}, "kanban": {true, false}, "tasks": {false, true}, "none": {false, false},
	} {
		c := DefaultConfig("")
		c.DoneTagHides = scope
		if c.DoneTagHidesKanban() != want[0] || c.DoneTagHidesTasks() != want[1] {
			t.Errorf("scope %q: kanban=%v tasks=%v, want %v", scope, c.DoneTagHidesKanban(), c.DoneTagHidesTasks(), want)
		}
	}
}

func TestInvalidDoneTagOptionsGoBackToDefaults(t *testing.T) {
	isolate(t)
	writeDiskConfig(t, `{"done_tag":"not a tag!","done_tag_hides":"sometimes","hide_completed_kanban":true}`)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DoneTag != "lm-done" || cfg.DoneTagHides != "both" {
		t.Errorf("done_tag = %q, done_tag_hides = %q; invalid values must go back to the defaults", cfg.DoneTag, cfg.DoneTagHides)
	}
	if !cfg.HideCompletedKanban {
		t.Error("a valid hide_completed_kanban must be kept")
	}
}

func TestDoneTagOptionsPersist(t *testing.T) {
	isolate(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.HideCompletedKanban, cfg.DoneTag, cfg.DoneTagHides = true, "archive", "kanban"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := Load(cfg.NotesDir)
	if err != nil {
		t.Fatal(err)
	}
	if !again.HideCompletedKanban || again.DoneTag != "archive" || again.DoneTagHides != "kanban" {
		t.Errorf("got %v %q %q", again.HideCompletedKanban, again.DoneTag, again.DoneTagHides)
	}
}
