package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// hideModel opens a model on two controlled notes: three finished tasks (gamma, delta, epsilon) and two open ones (alpha, beta).
// The interface is in English so the "N hidden" counter can be read.
func hideModel(t *testing.T, tweak func(m *AppModel)) (*AppModel, string) {
	t.Helper()
	m := newTestModel(t, 140, 40)
	dir := m.c.store.BaseDir
	for _, n := range m.c.notes {
		_ = os.Remove(n.Path)
	}
	a, b := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	if err := os.WriteFile(a, []byte("# A\n- [ ] alpha\n- [ ] beta #kb/doing\n- [x] gamma\n- [x] delta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("# B\n- [x] epsilon\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(b, old, old)
	if tweak != nil {
		tweak(m)
	}
	m.c.reload()
	m.afterChange()
	prev := i18n.CurrentLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(string(prev)) })
	return m, a
}

// inTasks reports whether the Tasks panel lists a task with word (the preview pane shows the whole note, so the screen is not a good witness).
func inTasks(m *AppModel, word string) bool {
	for _, tk := range m.c.tasks {
		if strings.Contains(tk.Text, word) {
			return true
		}
	}
	return false
}

func visible(m *AppModel, word string) bool { return strings.Contains(plain(m), word) }

func TestKanbanShowsFinishedCardsByDefault(t *testing.T) {
	m, _ := hideModel(t, nil)
	press(m, "W")
	for _, w := range []string{"alpha", "gamma", "delta", "epsilon"} {
		if !visible(m, w) {
			t.Errorf("%q must be visible by default", w)
		}
	}
	if visible(m, "hidden") {
		t.Error("nothing is hidden by default, so no counter")
	}
}

func TestKanbanOHidesFinishedCardsAndCountsThem(t *testing.T) {
	m, _ := hideModel(t, nil)
	press(m, "W", "o")
	for _, w := range []string{"gamma", "delta", "epsilon"} {
		if visible(m, w) {
			t.Errorf("%q is finished and must be hidden", w)
		}
	}
	for _, w := range []string{"alpha", "beta"} {
		if !visible(m, w) {
			t.Errorf("%q is open and must stay visible", w)
		}
	}
	if !visible(m, "3 hidden") {
		t.Errorf("the screen must say \"3 hidden\":\n%s", plain(m))
	}
	press(m, "o")
	if !visible(m, "gamma") || visible(m, "hidden") {
		t.Errorf("o again must show them and drop the counter:\n%s", plain(m))
	}
}

func TestKanbanHideFinishedFromConfigAndPersistence(t *testing.T) {
	m, _ := hideModel(t, func(m *AppModel) { m.c.cfg.HideCompletedKanban = true })
	press(m, "W")
	if visible(m, "gamma") || !visible(m, "3 hidden") {
		t.Fatalf("hide_completed_kanban=true must start hidden:\n%s", plain(m))
	}
	press(m, "o")
	if m.c.cfg.HideCompletedKanban {
		t.Error("x must save the new state in the config")
	}
}

func TestKanbanHideButtonClicks(t *testing.T) {
	m, _ := hideModel(t, nil)
	press(m, "W")
	x, y, ok := cellOf(m, "Hide done (o)")
	if !ok {
		t.Fatalf("the top bar needs a \"Hide done (o)\" button:\n%s", plain(m))
	}
	click(m, x+2, y)
	if visible(m, "gamma") {
		t.Error("clicking the button must hide the finished cards")
	}
}

func TestKanbanArchiveKeyTagsAndHidesTheCard(t *testing.T) {
	m, a := hideModel(t, nil)
	press(m, "W", "a") // the cursor starts on the first card of "To do": alpha
	body, _ := os.ReadFile(a)
	if !strings.Contains(string(body), "- [ ] alpha #lm-done\n") {
		t.Fatalf("a must add the done tag to the card's line, got:\n%s", body)
	}
	if visible(m, "alpha") || !visible(m, "1 hidden") {
		t.Fatalf("a tagged card is hidden and counted:\n%s", plain(m))
	}
	press(m, "A") // reveal
	if !visible(m, "alpha") || visible(m, "hidden") {
		t.Fatalf("A must reveal everything:\n%s", plain(m))
	}
	press(m, "a") // untag the card under the cursor (alpha again)
	body, _ = os.ReadFile(a)
	if strings.Contains(string(body), "#lm-done") {
		t.Fatalf("a on a tagged card must remove the tag, got:\n%s", body)
	}
	press(m, "A")
	if !visible(m, "alpha") {
		t.Error("an untagged card is visible again")
	}
}

func TestKanbanTagScopeNoneAndTasksOnly(t *testing.T) {
	for _, scope := range []string{"none", "tasks"} {
		m, _ := hideModel(t, func(m *AppModel) { m.c.cfg.DoneTagHides = scope })
		press(m, "W", "a")
		if !visible(m, "alpha") || visible(m, "hidden") {
			t.Errorf("done_tag_hides=%q must not hide in the Kanban:\n%s", scope, plain(m))
		}
	}
}

func TestTasksArchiveKeyTagsAndHidesTheTask(t *testing.T) {
	m, a := hideModel(t, nil)
	press(m, "2")
	i := taskIndex(t, m, "alpha")
	m.tasks.list.set(i, len(m.c.tasks))
	press(m, "a")
	body, _ := os.ReadFile(a)
	if !strings.Contains(string(body), "- [ ] alpha #lm-done\n") {
		t.Fatalf("a must tag the task's line, got:\n%s", body)
	}
	if inTasks(m, "alpha") || !visible(m, "1 hidden") {
		t.Fatalf("a tagged task is hidden and counted:\n%s", plain(m))
	}
	press(m, "A")
	if !inTasks(m, "alpha") {
		t.Errorf("A must reveal the tagged task:\n%s", plain(m))
	}
}

func TestTasksCounterForHiddenFinishedTasks(t *testing.T) {
	m, _ := hideModel(t, nil)
	press(m, "2", "H")
	if inTasks(m, "gamma") || !visible(m, "3 hidden") {
		t.Fatalf("H hides the finished tasks and the panel counts them:\n%s", plain(m))
	}
}

func TestTasksTagScopeKanbanOnlyKeepsThemVisible(t *testing.T) {
	m, _ := hideModel(t, func(m *AppModel) { m.c.cfg.DoneTagHides = "kanban" })
	press(m, "2")
	m.tasks.list.set(taskIndex(t, m, "alpha"), len(m.c.tasks))
	press(m, "a")
	if !inTasks(m, "alpha") || visible(m, "hidden") {
		t.Errorf("done_tag_hides=kanban must not hide in the Tasks panel:\n%s", plain(m))
	}
}
