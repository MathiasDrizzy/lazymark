package views

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

var mdImageRegex = regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

// RenderPreview renderiza el panel derecho de vista previa con Glamour (Markdown), imágenes inline y scroll vertical/horizontal
func RenderPreview(note *storage.Note, width, height int, active bool, kittyClient *image.Client, scrollY, scrollX int) string {
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

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(contentWidth),
		glamour.WithPreservedNewLines(),
	)

	// Intercalar texto Markdown e imágenes en la posición exacta donde aparecen en el documento
	content := note.Content
	matches := mdImageRegex.FindAllStringSubmatchIndex(content, -1)

	var renderedSections []string
	lastIdx := 0

	for _, m := range matches {
		textBefore := content[lastIdx:m[0]]
		altText := content[m[2]:m[3]]
		imgPath := content[m[4]:m[5]]
		lastIdx = m[1]

		// Renderizar el texto antes de la imagen con Glamour
		if strings.TrimSpace(textBefore) != "" {
			if renderer != nil {
				if out, err := renderer.Render(textBefore); err == nil {
					renderedSections = append(renderedSections, strings.TrimRight(out, "\n"))
				} else {
					renderedSections = append(renderedSections, textBefore)
				}
			} else {
				renderedSections = append(renderedSections, textBefore)
			}
		}

		// Resolver e insertar la imagen inline con colores reales
		resolvedImg := imgPath
		if !filepath.IsAbs(resolvedImg) {
			resolvedImg = filepath.Join(filepath.Dir(note.Path), resolvedImg)
		}

		if ansiImg, err := image.RenderInlineToAnsi(resolvedImg, contentWidth-4, 12); err == nil && ansiImg != "" {
			caption := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true).
				Render(fmt.Sprintf("   %s (%s)", altText, filepath.Base(imgPath)))
			renderedSections = append(renderedSections, fmt.Sprintf("%s\n%s", caption, ansiImg))
		}
	}

	// Renderizar el texto restante después de la última imagen
	if lastIdx < len(content) {
		remaining := content[lastIdx:]
		if strings.TrimSpace(remaining) != "" {
			if renderer != nil {
				if out, err := renderer.Render(remaining); err == nil {
					renderedSections = append(renderedSections, strings.TrimRight(out, "\n"))
				} else {
					renderedSections = append(renderedSections, remaining)
				}
			} else {
				renderedSections = append(renderedSections, remaining)
			}
		}
	}

	finalContent := strings.Join(renderedSections, "\n\n")
	if len(renderedSections) == 0 {
		finalContent = content
	}

	lines := strings.Split(finalContent, "\n")
	totalLines := len(lines)
	usableHeight := height - 2
	if usableHeight < 1 {
		usableHeight = 1
	}

	if scrollY > totalLines-usableHeight {
		scrollY = totalLines - usableHeight
	}
	if scrollY < 0 {
		scrollY = 0
	}

	endLine := scrollY + usableHeight
	if endLine > totalLines {
		endLine = totalLines
	}

	var visibleLines []string
	for i := scrollY; i < endLine; i++ {
		line := lines[i]
		if scrollX > 0 {
			line = ansi.CutWc(line, scrollX, scrollX+contentWidth)
		}
		visibleLines = append(visibleLines, line)
	}

	// Asegurar que el panel tenga usableHeight líneas para fijar el indicador en la esquina inferior
	for len(visibleLines) < usableHeight {
		visibleLines = append(visibleLines, "")
	}

	// Indicador de posición fijo en la esquina inferior derecha del panel
	if totalLines > usableHeight && len(visibleLines) > 0 {
		pct := (scrollY * 100) / (totalLines - usableHeight)
		badgeStr := fmt.Sprintf("[%d%% • %d/%d]", pct, scrollY+1, totalLines)
		badge := theme.NormalItem.Copy().Foreground(theme.ColorPeach).Bold(true).Render(badgeStr)
		badgeW := ansi.StringWidth(badgeStr)

		lastIdx := usableHeight - 1
		lastLine := visibleLines[lastIdx]
		lastLineWidth := ansi.StringWidth(lastLine)

		if lastLineWidth+badgeW+1 <= contentWidth {
			padding := strings.Repeat(" ", contentWidth-lastLineWidth-badgeW)
			visibleLines[lastIdx] = lastLine + padding + badge
		} else {
			truncated := ansi.Truncate(lastLine, contentWidth-badgeW-2, "")
			padding := strings.Repeat(" ", contentWidth-ansi.StringWidth(truncated)-badgeW)
			visibleLines[lastIdx] = truncated + padding + badge
		}
	}

	viewContent := strings.Join(visibleLines, "\n")

	return borderStyle.
		Width(width).
		Height(height).
		Render(viewContent)
}
