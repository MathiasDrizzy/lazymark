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

	// Renderizar imágenes inline integradas directamente en la nota (estilo Apple Notes)
	if len(note.Images) > 0 {
		var imgSections []string
		for _, imgPath := range note.Images {
			resolvedImg := imgPath
			if !filepath.IsAbs(resolvedImg) {
				resolvedImg = filepath.Join(filepath.Dir(note.Path), resolvedImg)
			}
			if ansiImg, err := image.RenderInlineToAnsi(resolvedImg, contentWidth-4, 12); err == nil && ansiImg != "" {
				caption := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true).
					Render(fmt.Sprintf("  🖼️ %s", filepath.Base(imgPath)))
				imgSections = append(imgSections, fmt.Sprintf("%s\n%s", caption, ansiImg))
			}
		}
		if len(imgSections) > 0 {
			renderedContent = fmt.Sprintf("%s\n\n%s", renderedContent, strings.Join(imgSections, "\n\n"))
		}
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(renderedContent)
}
