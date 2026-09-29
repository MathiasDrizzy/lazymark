package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/glamour"
)

// RenderPreview renderiza el panel derecho de vista previa con Glamour (Markdown) o Kitty Graphics
func RenderPreview(note *storage.Note, width, height int, active bool, kittyClient *image.Client) string {
	borderStyle := theme.InactivePanelBorder
	if active {
		borderStyle = theme.ActivePanelBorder
	}

	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	if note == nil {
		empty := theme.NormalItem.Copy().Italic(true).Render(i18n.T("Selecciona una nota para ver el contenido...", "Select a note to view content..."))
		return borderStyle.Width(width).Height(height).Render(empty)
	}

	// Renderizar Markdown con Glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(contentWidth),
	)

	var renderedContent string
	if err == nil {
		out, err := renderer.Render(note.Content)
		if err == nil {
			renderedContent = out
		} else {
			renderedContent = note.Content
		}
	} else {
		renderedContent = note.Content
	}

	// Si la nota tiene imágenes adjuntas, añadir badges limpios sin secuencias de escape corruptoras
	if len(note.Images) > 0 {
		var imgBadges []string
		for _, imgPath := range note.Images {
			imgBadges = append(imgBadges, theme.TagBadge.Render("🖼️ "+filepath.Base(imgPath)))
		}
		attachmentLabel := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
			Render(i18n.T("Adjuntos (ver en pestaña Galería):", "Attachments (view in Gallery tab):"))
		renderedContent = fmt.Sprintf("%s\n\n  %s %s", renderedContent, attachmentLabel, strings.Join(imgBadges, " "))
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(renderedContent)
}
