package views

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
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

// RenderTagList genera el panel izquierdo con la lista de tags y registra clics
func RenderTagList(tags []TagInfo, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	title := i18n.T("[3] Categorías", "[3] Categories")
	badge := "0 of 0"
	if len(tags) > 0 {
		badge = fmt.Sprintf("%d of %d", selectedIndex+1, len(tags))
	}

	if len(tags) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(i18n.T("  (No hay tags. Usa #tag en tus notas)", "  (No tags yet. Use #tag in your notes)"))
		rows = append(rows, emptyMsg)
	}

	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}

	startIdx := 0
	if selectedIndex >= usableHeight {
		startIdx = selectedIndex - usableHeight + 1
	}
	endIdx := startIdx + usableHeight
	if endIdx > len(tags) {
		endIdx = len(tags)
	}

	contentWidth := width - 4
	if contentWidth < 4 {
		contentWidth = 4
	}

	for i := startIdx; i < endIdx; i++ {
		tag := tags[i]
		isSelected := i == selectedIndex

		var selStyle lipgloss.Style
		if isSelected {
			if active {
				selStyle = theme.SelectedLineActive
			} else {
				selStyle = theme.SelectedLineInactive
			}
		}

		var rowText string
		if isSelected {
			cursor := "▸ "
			rawLine := fmt.Sprintf("%s#%s (%d)", cursor, tag.Name, tag.NoteCount)
			rawLine = ansi.Truncate(rawLine, contentWidth, "")
			lineW := ansi.StringWidth(rawLine)
			if lineW < contentWidth {
				rawLine += strings.Repeat(" ", contentWidth-lineW)
			}
			rowText = selStyle.Render(rawLine)
		} else {
			cursor := "  "
			tagBadge := theme.TagBadge.Render(fmt.Sprintf("#%s", tag.Name))
			countStr := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%d)", tag.NoteCount))
			rowText = fmt.Sprintf("%s%s %s", cursor, tagBadge, countStr)
			rowText = ansi.Truncate(rowText, contentWidth, "")
		}

		// Registrar zona de clic para esta fila
		if ht != nil {
			rowY := offsetY + 1 + (i - startIdx)
			ht.Register(fmt.Sprintf("tag-%d", i), mouse.ZoneTag, 0, rowY, width, rowY, i, tag.Name)
		}

		rows = append(rows, rowText)
	}

	content := strings.Join(rows, "\n")
	return theme.RenderBoxWithTitle(title, badge, content, width, height, active)
}

// RenderTagPreview muestra las notas asociadas al tag seleccionado
func RenderTagPreview(notes []storage.Note, tag string, width, height int, active bool) string {
	title := i18n.T("[4] Vista Previa", "[4] Preview")
	badge := ""

	if tag == "" {
		empty := theme.NormalItem.Copy().Italic(true).Render(i18n.T("Selecciona un tag para ver sus notas...", "Select a tag to view its notes..."))
		return theme.RenderBoxWithTitle(title, badge, empty, width, height, active)
	}

	badge = "#" + tag

	filtered := NotesForTag(notes, tag)

	headerStyle := theme.SelectedItem.Copy()
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
		if len(title) > maxLen {
			title = title[:maxLen-3] + "..."
		}

		noteRow := fmt.Sprintf("    %s  %s",
			theme.NormalItem.Render(title),
			theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(timeStr),
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
