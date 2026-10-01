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

	cornerTL := borderStyle.Render("┌")
	cornerTR := borderStyle.Render("┐")
	cornerBL := borderStyle.Render("└")
	cornerBR := borderStyle.Render("┘")
	borderVLeft := borderStyle.Render("│ ")
	borderVRight := borderStyle.Render(" │")

	// 2. Construir Borde Superior con Título a la Izquierda y Contador a la Extrema Derecha
	// Formato: ┌─ [N] Titulo ────────────────────────────────────────── 1 of 6 ─┐
	var topRow string
	availTop := width - 2 // Descontando ┌ y ┐

	if title == "" && badge == "" {
		topRow = cornerTL + borderStyle.Render(strings.Repeat("─", availTop)) + cornerTR
	} else {
		titleStr := title
		badgeStr := badge

		titleLen := ansi.StringWidth(titleStr)
		badgeLen := ansi.StringWidth(badgeStr)

		// Verificar si caben título y badge con al menos 1 guion intermedio
		// overhead: cornerTL(1) + "─ "(2) + " "(1) + " "(1) + " ─"(2) + cornerTR(1) = 8 + 1 guion = 9
		if badgeStr != "" {
			minNeeded := titleLen + badgeLen + 9
			if width < minNeeded {
				// Espacio insuficiente: intentar truncar el título preservando el badge si es posible
				availForTitle := width - badgeLen - 9
				if availForTitle >= 4 {
					titleStr = ansi.Truncate(titleStr, availForTitle, "…")
					titleLen = ansi.StringWidth(titleStr)
				} else {
					// Si es demasiado estrecho, omitir badge y truncar título
					badgeStr = ""
					badgeLen = 0
					availForTitleOnly := width - 6
					if availForTitleOnly >= 2 && titleLen > availForTitleOnly {
						titleStr = ansi.Truncate(titleStr, availForTitleOnly, "…")
						titleLen = ansi.StringWidth(titleStr)
					}
				}
			}
		} else if titleStr != "" {
			availForTitleOnly := width - 6
			if availForTitleOnly >= 2 && titleLen > availForTitleOnly {
				titleStr = ansi.Truncate(titleStr, availForTitleOnly, "…")
				titleLen = ansi.StringWidth(titleStr)
			}
		}

		// Construir parte izquierda
		var leftPart string
		var leftLen int
		if titleStr != "" {
			styledTitle := titleStyle.Render(titleStr)
			leftPart = cornerTL + borderStyle.Render("─ ") + styledTitle + borderStyle.Render(" ")
			leftLen = 1 + 2 + titleLen + 1
		} else {
			leftPart = cornerTL
			leftLen = 1
		}

		// Construir parte derecha
		var rightPart string
		var rightLen int
		if badgeStr != "" {
			styledBadge := badgeStyle.Render(badgeStr)
			rightPart = borderStyle.Render(" ") + styledBadge + borderStyle.Render(" ─") + cornerTR
			rightLen = 1 + badgeLen + 2 + 1
		} else {
			rightPart = cornerTR
			rightLen = 1
		}

		// Guiones intermedios
		middleLen := width - leftLen - rightLen
		if middleLen < 0 {
			middleLen = 0
		}
		dashes := borderStyle.Render(strings.Repeat("─", middleLen))
		topRow = leftPart + dashes + rightPart
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
