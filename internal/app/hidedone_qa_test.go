package app

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// hideWords are the five tasks of hideModel and whether each one starts finished.
var hideWords = []struct {
	word string
	done bool
}{{"alpha", false}, {"beta", false}, {"gamma", true}, {"delta", true}, {"epsilon", true}}

// taggedOnDisk reports which of the words carry the done tag in the notes, read from the files and not from the app.
func taggedOnDisk(t *testing.T, m *AppModel) map[string]bool {
	t.Helper()
	tagged := map[string]bool{}
	for _, name := range []string{"a.md", "b.md"} {
		body, err := os.ReadFile(m.c.store.BaseDir + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(body), "\n") {
			for _, w := range hideWords {
				if strings.Contains(line, w.word) && strings.Contains(line, "#"+m.c.cfg.DoneTag) {
					tagged[w.word] = true
				}
			}
		}
	}
	return tagged
}

// savedHide reads the hide flags back from config.json, which is what the next run will start from.
func savedHide(t *testing.T, m *AppModel) (kanban, tasks bool) {
	t.Helper()
	body, err := os.ReadFile(m.c.cfg.Path())
	if err != nil {
		return false, false
	}
	var raw struct {
		Kanban bool `json:"hide_completed_kanban"`
		Tasks  bool `json:"hide_completed_tasks"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	return raw.Kanban, raw.Tasks
}

// sequences lists every sequence of up to maxLen keys drawn from keys.
func sequences(keys []string, maxLen int) [][]string {
	out := [][]string{{}}
	frontier := [][]string{{}}
	for n := 0; n < maxLen; n++ {
		var next [][]string
		for _, s := range frontier {
			for _, k := range keys {
				seq := append(append([]string{}, s...), k)
				next = append(next, seq)
			}
		}
		out = append(out, next...)
		frontier = next
	}
	return out
}

// TestKanbanHideRevealTagQA runs every sequence of o, a and A on the board and, after each key, compares the screen and config.json
// with a model that knows only three rules: o flips "hide finished" and ends any reveal, A flips the reveal, and a card is hidden when
// it is finished and hiding is on, or when it carries the done tag, unless everything is revealed. The model was written after the report
// "after A, o stops working": the old code failed on the sequence A, o.
func TestKanbanHideRevealTagQA(t *testing.T) {
	for _, seq := range sequences([]string{"o", "a", "A"}, 4) {
		m, _ := hideModel(t, nil)
		press(m, "W")
		hide, reveal := false, false
		for i, k := range seq {
			press(m, k)
			switch k {
			case "o":
				hide, reveal = !hide, false
			case "A":
				reveal = !reveal
			}
			tagged := taggedOnDisk(t, m)
			for _, w := range hideWords {
				want := reveal || !((hide && w.done) || tagged[w.word])
				if got := visible(m, w.word); got != want {
					t.Fatalf("keys %v (after %q): %q visible=%v, want %v (hide=%v reveal=%v tagged=%v)\n%s",
						seq[:i+1], k, w.word, got, want, hide, reveal, tagged, plain(m))
				}
			}
			if kb, _ := savedHide(t, m); contains(seq[:i+1], "o") && kb != hide {
				t.Fatalf("keys %v: config.json says hide_completed_kanban=%v, want %v", seq[:i+1], kb, hide)
			}
		}
	}
}

// TestTasksHideRevealTagQA is the same for the Tasks panel with H (hide finished), a and A. There the finished tasks leave through the
// pending filter and the tagged ones through the done tag, so a reveal only brings back the tagged ones.
func TestTasksHideRevealTagQA(t *testing.T) {
	for _, seq := range sequences([]string{"H", "a", "A"}, 4) {
		m, _ := hideModel(t, nil)
		press(m, "2")
		hide, reveal := false, false
		for i, k := range seq {
			if k == "a" && len(m.c.tasks) > 0 {
				m.tasks.list.set(0, len(m.c.tasks)) // the first visible task, so the key always has a target
			}
			press(m, k)
			switch k {
			case "H":
				hide, reveal = !hide, false
			case "A":
				reveal = !reveal
			}
			tagged := taggedOnDisk(t, m)
			for _, w := range hideWords {
				want := !(hide && w.done) && (reveal || !tagged[w.word])
				if got := inTasks(m, w.word); got != want {
					t.Fatalf("keys %v (after %q): task %q listed=%v, want %v (hide=%v reveal=%v tagged=%v)",
						seq[:i+1], k, w.word, got, want, hide, reveal, tagged)
				}
			}
			if _, ts := savedHide(t, m); contains(seq[:i+1], "H") && ts != hide {
				t.Fatalf("keys %v: config.json says hide_completed_tasks=%v, want %v", seq[:i+1], ts, hide)
			}
		}
	}
}

func pressed(s []string, k string) bool {
	for _, v := range s {
		if v == k {
			return true
		}
	}
	return false
}
