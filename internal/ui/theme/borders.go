package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// RenderBoxWithTitle genera un panel rectangular estilo Lazygit con el título y badge embebidos
// directamente en el borde superior: ╭─ [N] Título (Badge) ────────╮
func RenderBoxWithTitle(title, badge, content string, width, height int, active bool) string {
	if width < 10 {
		width = 10
	}
	if height < 3 {
		height = 3
	}

	// 1. Estilos según estado de foco (Activo = Verde Lazygit, Inactivo = Surface1)
	borderCol := ColorSurface1
	titleCol := ColorSubtext0
	badgeCol := ColorOverlay0
	isBold := false

	if active {
		borderCol = ColorGreen
		titleCol = ColorGreen
		badgeCol = ColorPeach
		isBold = true
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderCol)
	titleStyle := lipgloss.NewStyle().Foreground(titleCol).Bold(isBold)
	badgeStyle := lipgloss.NewStyle().Foreground(badgeCol).Bold(isBold)

	cornerTL := borderStyle.Render("╭")
	cornerTR := borderStyle.Render("╮")
	cornerBL := borderStyle.Render("╰")
	cornerBR := borderStyle.Render("╯")
	borderVLeft := borderStyle.Render("│ ")
	borderVRight := borderStyle.Render(" │")

	// 2. Construir Borde Superior con Título Embebido
	var topRow string
	availTop := width - 2 // Descontando ╭ y ╮

	if title == "" {
		topRow = cornerTL + borderStyle.Render(strings.Repeat("─", availTop)) + cornerTR
	} else {
		// Formato: ╭─ [N] Titulo (Badge) ─────╮
		// Prefijo ─  (2 columnas)
		prefix := borderStyle.Render("─ ")
		usedLen := 2 // prefix

		styledTitle := titleStyle.Render(title)
		titleLen := ansi.StringWidth(title)

		styledBadge := ""
		badgeLen := 0
		if badge != "" {
			styledBadge = " " + badgeStyle.Render(badge)
			badgeLen = 1 + ansi.StringWidth(badge)
		}

		totalHeaderLen := titleLen + badgeLen
		maxHeaderLen := availTop - 4 // Dejar espacio para prefix (2), sufijo " " (1) y al menos 1 guion
		if maxHeaderLen < 4 {
			maxHeaderLen = 4
		}

		if totalHeaderLen > maxHeaderLen {
			// Truncar si el título excede el espacio superior disponible
			if badgeLen > 0 && maxHeaderLen > badgeLen+3 {
				truncTitle := ansi.Truncate(title, maxHeaderLen-badgeLen, "…")
				styledTitle = titleStyle.Render(truncTitle)
				titleLen = ansi.StringWidth(truncTitle)
			} else {
				styledBadge = ""
				badgeLen = 0
				truncTitle := ansi.Truncate(title, maxHeaderLen, "…")
				styledTitle = titleStyle.Render(truncTitle)
				titleLen = ansi.StringWidth(truncTitle)
			}
		}

		headerContent := styledTitle + styledBadge
		usedLen += titleLen + badgeLen + 1 // +1 por el espacio posterior " "

		remainingDashes := availTop - usedLen
		if remainingDashes < 1 {
			remainingDashes = 1
		}

		// Rellenar con guiones hasta completar exactamente availTop
		// Compensar diferencias por redondeo
		currentLen := 2 + (titleLen + badgeLen) + 1 + remainingDashes
		if currentLen < availTop {
			remainingDashes += availTop - currentLen
		} else if currentLen > availTop {
			remainingDashes -= (currentLen - availTop)
			if remainingDashes < 0 {
				remainingDashes = 0
			}
		}

		dashes := borderStyle.Render(strings.Repeat("─", remainingDashes))
		topRow = cornerTL + prefix + headerContent + " " + dashes + cornerTR
	}

	// 3. Procesar Contenido Interior
	contentWidth := width - 4 // 2 de borderVLeft ("│ ") y 2 de borderVRight (" │")
	if contentWidth < 1 {
		contentWidth = 1
	}
	contentHeight := height - 2 // Descontando borde superior e inferior

	contentLines := strings.Split(content, "\n")
	var bodyRows []string

	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Truncar o rellenar la línea para que mida exactamente contentWidth
		lineW := ansi.StringWidth(line)
		if lineW > contentWidth {
			line = ansi.Truncate(line, contentWidth, "")
			lineW = ansi.StringWidth(line)
		}
		if lineW < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-lineW)
		}
		bodyRows = append(bodyRows, borderVLeft+line+borderVRight)
	}

	// 4. Borde Inferior
	bottomRow := cornerBL + borderStyle.Render(strings.Repeat("─", availTop)) + cornerBR

	// 5. Ensamblaje final
	allRows := make([]string, 0, height)
	allRows = append(allRows, topRow)
	allRows = append(allRows, bodyRows...)
	allRows = append(allRows, bottomRow)

	return strings.Join(allRows, "\n")
}
