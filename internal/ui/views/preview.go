package views

import (
	"fmt"
	"strings"

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
		empty := theme.NormalItem.Copy().Italic(true).Render("Selecciona una nota para ver el contenido...")
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

	// Si la nota tiene imágenes adjuntas, añadir la representación gráfica Kitty
	if len(note.Images) > 0 && kittyClient != nil && kittyClient.Supported {
		var imgOutputs []string
		for _, imgPath := range note.Images {
			imgOutputs = append(imgOutputs, kittyClient.RenderCommand(imgPath, contentWidth, 12))
		}
		renderedContent = fmt.Sprintf("%s\n\n%s", renderedContent, strings.Join(imgOutputs, "\n"))
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(renderedContent)
}
