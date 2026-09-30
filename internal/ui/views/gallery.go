package views

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/image"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/theme"
)

// ImageEntry representa una imagen aplanada con contexto de la nota padre
type ImageEntry struct {
	Path      string
	AltText   string
	NoteTitle string
	NotePath  string
}

// CollectImages extrae todas las referencias de imagen de todas las notas
func CollectImages(notes []storage.Note) []ImageEntry {
	var images []ImageEntry
	for _, note := range notes {
		for _, imgPath := range note.Images {
			alt := filepath.Base(imgPath)
			images = append(images, ImageEntry{
				Path:      imgPath,
				AltText:   alt,
				NoteTitle: note.Title,
				NotePath:  note.Path,
			})
		}
	}
	return images
}

// RenderGalleryList genera el panel izquierdo con la lista de imágenes/adjuntos y registra clics
func RenderGalleryList(images []ImageEntry, selectedIndex int, width, height int, active bool, ht *mouse.HitTester, offsetY int) string {
	var rows []string

	// Cabecera
	header := theme.SelectedItem.Copy().Foreground(theme.ColorBlue).
		Render(fmt.Sprintf(" 🖼️  %s (%d %s)", i18n.T("Galería", "Gallery"), len(images), i18n.T("adjuntos", "attachments")))
	rows = append(rows, header)

	if len(images) == 0 {
		emptyMsg := theme.NormalItem.Copy().Italic(true).Render(i18n.T("  (Sin imágenes. Usa 'p' para pegar del portapapeles)", "  (No images. Press 'p' to paste from clipboard)"))
		rows = append(rows, emptyMsg)
	}

	usableHeight := height - 3 // -2 border -1 header
	if usableHeight < 1 {
		usableHeight = 1
	}

	startIdx := 0
	if selectedIndex >= usableHeight {
		startIdx = selectedIndex - usableHeight + 1
	}
	endIdx := startIdx + usableHeight
	if endIdx > len(images) {
		endIdx = len(images)
	}

	for i := startIdx; i < endIdx; i++ {
		img := images[i]
		isSelected := i == selectedIndex

		cursor := "  "
		if isSelected {
			cursor = theme.SelectedItem.Render("❯ ")
		}

		// Icono según extensión
		ext := strings.ToLower(filepath.Ext(img.Path))
		icon := "🖼️"
		switch ext {
		case ".png":
			icon = "📸"
		case ".jpg", ".jpeg":
			icon = "🏞️"
		case ".gif":
			icon = "🎞️"
		case ".svg":
			icon = "✏️"
		case ".pdf":
			icon = "📄"
		}

		// Nombre truncado
		maxNameLen := width - 18
		if maxNameLen < 5 {
			maxNameLen = 5
		}
		fileName := filepath.Base(img.Path)
		if len(fileName) > maxNameLen {
			fileName = fileName[:maxNameLen-3] + "..."
		}

		var fileStyle string
		if isSelected {
			fileStyle = theme.SelectedItem.Render(fileName)
		} else {
			fileStyle = theme.NormalItem.Render(fileName)
		}

		// Nota origen
		noteRef := theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(fmt.Sprintf("(%s)", img.NoteTitle))

		rowText := fmt.Sprintf("%s%s %s %s", cursor, icon, fileStyle, noteRef)

		// Registrar zona de clic
		if ht != nil {
			rowY := offsetY + (i - startIdx) + 2 // +1 border +1 header
			ht.Register(fmt.Sprintf("gallery-%d", i), mouse.ZoneGallery, 0, rowY, width, rowY, i, img.Path)
		}

		rows = append(rows, rowText)
	}

	content := strings.Join(rows, "\n")

	borderStyle := theme.InactivePanelBorder
	if active {
		borderStyle = theme.ActivePanelBorder
	}

	return borderStyle.
		Width(width).
		Height(height).
		Render(content)
}

// RenderGalleryPreview muestra la vista previa de la imagen seleccionada con Kitty Graphics o fallback
func RenderGalleryPreview(entry *ImageEntry, width, height int, active bool, kittyClient *image.Client) string {
	borderStyle := theme.InactivePanelBorder
	if active {
		borderStyle = theme.ActivePanelBorder
	}

	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	if entry == nil {
		empty := theme.NormalItem.Copy().Italic(true).Render(i18n.T("Selecciona una imagen para previsualizarla...", "Select an image to preview..."))
		return borderStyle.Width(width).Height(height).Render(empty)
	}

	var rows []string

	// Metadatos del archivo
	headerStyle := theme.SelectedItem.Copy().Foreground(theme.ColorBlue)
	rows = append(rows, headerStyle.Render(fmt.Sprintf("  󰋩  %s", filepath.Base(entry.Path))))
	rows = append(rows, "")

	// Información
	noteRef := fmt.Sprintf("    %s %s", i18n.T("Nota:", "Note:"), theme.NormalItem.Render(entry.NoteTitle))
	rows = append(rows, noteRef)

	pathRef := fmt.Sprintf("    %s %s",
		i18n.T("Ruta:", "Path:"),
		theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Render(entry.Path))
	rows = append(rows, pathRef)

	ext := strings.ToLower(filepath.Ext(entry.Path))
	formatRef := fmt.Sprintf("    %s %s",
		i18n.T("Formato:", "Format:"),
		theme.NormalItem.Copy().Foreground(theme.ColorTeal).Render(strings.ToUpper(strings.TrimPrefix(ext, "."))))
	rows = append(rows, formatRef)
	rows = append(rows, "")

	resolvedPath := entry.Path
	if !filepath.IsAbs(resolvedPath) && entry.NotePath != "" {
		resolvedPath = filepath.Join(filepath.Dir(entry.NotePath), resolvedPath)
	}

	// Renderizado Kitty o Fallback
	if kittyClient != nil && kittyClient.Supported {
		imgRows := height - 12
		if imgRows > 12 {
			imgRows = 12
		}
		if imgRows < 6 {
			imgRows = 6
		}
		imgCols := contentWidth - 4
		if imgCols > 40 {
			imgCols = 40
		}
		if imgCols < 10 {
			imgCols = 10
		}
		imgOutput := kittyClient.RenderCommand(resolvedPath, imgCols, imgRows)
		rows = append(rows, imgOutput)
	} else {
		// Fallback elegante para terminales sin soporte gráfico
		box := strings.Repeat("─", contentWidth-2)
		rows = append(rows, fmt.Sprintf("  ┌%s┐", box))
		artHeight := height - 14
		if artHeight < 3 {
			artHeight = 3
		}
		for range artHeight {
			padding := strings.Repeat(" ", contentWidth-2)
			rows = append(rows, fmt.Sprintf("  │%s│", padding))
		}
		centerLine := fmt.Sprintf("  │%s│",
			theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).
				Render(fmt.Sprintf("%*s🖼️ %s%*s",
					(contentWidth-len(filepath.Base(entry.Path))-6)/2, "",
					filepath.Base(entry.Path),
					(contentWidth-len(filepath.Base(entry.Path))-6)/2, "",
				)))
		rows = append(rows, centerLine)
		rows = append(rows, fmt.Sprintf("  └%s┘", box))
		rows = append(rows, "")
		rows = append(rows, theme.NormalItem.Copy().Foreground(theme.ColorOverlay0).Italic(true).
			Render(i18n.T("  (Terminal sin soporte Kitty Graphics)", "  (Terminal without Kitty Graphics support)")))
	}

	content := strings.Join(rows, "\n")
	return borderStyle.Width(width).Height(height).Render(content)
}
