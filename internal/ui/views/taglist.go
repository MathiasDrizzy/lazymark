package views

import (
	"fmt"
	"github.com/MathiasDrizzy/lazymark/internal/ui/textwidth"
	"sort"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// TagInfo agrupa un tag con las notas que lo contienen
type TagInfo struct {
	Name      string
	NoteCount int
}

// CollectTags extrae todos los tags únicos de las notas con su frecuencia
func CollectTags(notes []storage.Note) []TagInfo {
	tagCount := make(map[string]int)
	for _, note := range notes {
		for _, tag := range note.Tags {
			tagCount[tag]++
		}
	}

	var tags []TagInfo
	for name, count := range tagCount {
		tags = append(tags, TagInfo{Name: name, NoteCount: count})
	}

	// Ordenar alfabéticamente
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags
}

// NotesForTag devuelve las notas que contienen un tag específico
func NotesForTag(notes []storage.Note, tag string) []storage.Note {
	var filtered []storage.Note
	for _, note := range notes {
		for _, t := range note.Tags {
			if t == tag {
				filtered = append(filtered, note)
				break
			}
		}
	}
	return filtered
}

// RenderTagPreview muestra las notas asociadas al tag seleccionado
func RenderTagPreview(notes []storage.Note, tag string, width, height int, active bool) string {
	title := i18n.T("[4] Vista Previa", "[4] Preview")
	badge := ""

	if tag == "" {
		empty := theme.NormalItem.Italic(true).Render(i18n.T("Selecciona un tag para ver sus notas...", "Select a tag to view its notes..."))
		return theme.RenderBoxWithTitle(title, badge, empty, width, height, active)
	}

	badge = "#" + tag

	filtered := NotesForTag(notes, tag)

	headerStyle := theme.SelectedItem
	header := headerStyle.Render(fmt.Sprintf("  #%s — %d %s", tag, len(filtered), i18n.T("nota(s)", "note(s)")))

	var rows []string
	rows = append(rows, header)
	rows = append(rows, strings.Repeat("─", width-4))

	for _, note := range filtered {
		timeStr := note.ModTime.Format("02 Jan 15:04")
		title := note.Title

		maxLen := width - 20
		if maxLen < 5 {
			maxLen = 5
		}
		title = textwidth.Truncate(title, maxLen, textwidth.Ellipsis)

		noteRow := fmt.Sprintf("    %s  %s",
			theme.NormalItem.Render(title),
			theme.NormalItem.Foreground(theme.ColorOverlay0).Render(timeStr),
		)
		rows = append(rows, noteRow)

		// Mostrar tags de la nota
		if len(note.Tags) > 0 {
			var tagBadges []string
			for _, t := range note.Tags {
				tagBadges = append(tagBadges, theme.TagBadge.Render("#"+t))
			}
			rows = append(rows, "     "+strings.Join(tagBadges, " "))
		}
	}

	content := strings.Join(rows, "\n")
	return theme.RenderBoxWithTitle(title, badge, content, width, height, active)
}
