package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

// kanbanSheet es la Hoja 2: el tablero con las tareas, en las columnas configuradas.
type kanbanSheet struct {
	c        *core
	col      int
	selected []int
	press    *kanbanPress // el botón del mouse está apretado sobre una tarjeta
	drag     views.KanbanDrag
}

// kanbanPress es el botón apretado sobre una tarjeta: si el puntero se mueve a otra columna con el botón apretado,
// es un arrastre; si se suelta donde empezó, fue un clic.
type kanbanPress struct {
	col, idx int
}

func (k *kanbanSheet) numCols() int { return k.c.board.NumCols() }

func (k *kanbanSheet) cards() []views.KanbanCard { return k.c.board.ColumnCards(k.col) }

func (k *kanbanSheet) current() *views.KanbanCard {
	cards := k.cards()
	if len(cards) == 0 {
		return nil
	}
	i := clamp(k.selection(k.col), 0, len(cards)-1)
	return &cards[i]
}

// selection devuelve la tarjeta seleccionada de la columna c (0 si no hay).
func (k *kanbanSheet) selection(c int) int {
	if c < 0 || c >= len(k.selected) {
		return 0
	}
	return k.selected[c]
}

// clampSelection mantiene un cursor por columna, cada uno dentro de sus tarjetas (también si cambió la cantidad de columnas).
func (k *kanbanSheet) clampSelection() {
	n := k.numCols()
	for len(k.selected) < n {
		k.selected = append(k.selected, 0)
	}
	k.selected = k.selected[:n]
	for c := range k.selected {
		k.selected[c] = clamp(k.selected[c], 0, max(0, len(k.c.board.ColumnCards(c))-1))
	}
	k.col = clamp(k.col, 0, max(0, n-1))
}

func (k *kanbanSheet) key(a Action) tea.Cmd {
	n := k.numCols()
	switch a {
	case actLeft:
		k.col = (k.col + n - 1) % n
	case actRight, actNextPanel:
		k.col = (k.col + 1) % n
	case actPanelNotes, actPanelTasks, actPanelTags:
		if c := int(a - actPanelNotes); c < n {
			k.col = c
		}
	case actUp:
		k.selected[k.col] = max(0, k.selected[k.col]-1)
	case actDown:
		k.selected[k.col] = min(max(0, len(k.cards())-1), k.selected[k.col]+1)
	case actMoveCardLeft:
		return k.moveTo(k.col - 1)
	case actMoveCardRight:
		return k.moveTo(k.col + 1)
	case actToggleTask:
		if card := k.current(); card != nil {
			target := k.c.cols().DoneIndex()
			if card.Column == target {
				target = 0
			}
			return k.setColumn(card, target)
		}
	case actEdit:
		if card := k.current(); card != nil {
			return k.c.openEditor(card.NotePath, card.Task.Line)
		}
	}
	return nil
}

func (k *kanbanSheet) moveTo(target int) tea.Cmd {
	if card := k.current(); card != nil && target >= 0 && target < k.numCols() {
		return k.setColumn(card, target)
	}
	return nil
}

// noteTime es el mtime con el que se cargó la nota (la comprobación de X10 al escribir).
func (k *kanbanSheet) noteTime(path string) time.Time {
	for _, n := range k.c.notes {
		if n.Path == path {
			return n.ModTime
		}
	}
	return time.Time{}
}

// setColumn reescribe solo la línea de la tarjeta (casilla y tag) y deja el foco en la columna destino, con el cursor
// sobre la tarjeta movida. Si la nota cambió en disco desde que se cargó, no la pisa: recarga y avisa.
func (k *kanbanSheet) setColumn(card *views.KanbanCard, target int) tea.Cmd {
	if target == card.Column {
		return nil
	}
	path, line := card.NotePath, card.Task.Line
	err := k.c.store.MoveTask(path, line, k.c.cols(), target, k.noteTime(path))
	switch {
	case errors.Is(err, storage.ErrNoteChanged):
		k.c.reload()
		k.clampSelection()
		k.c.setStatus("%s", i18n.T("Cambió por fuera: recargada", "Changed outside: reloaded"))
		return nil
	case err != nil:
		k.c.errStatus("No se pudo mover la tarjeta", "Could not move card", err)
		return nil
	}
	k.c.reload()
	k.clampSelection()
	k.col = target
	// el cursor sigue a la tarjeta movida (la misma nota y línea), no a la última de la columna
	k.selected[target] = max(0, len(k.c.board.ColumnCards(target))-1)
	for i, c := range k.c.board.ColumnCards(target) {
		if c.NotePath == path && c.Task.Line == line {
			k.selected[target] = i
			break
		}
	}
	k.c.setStatus("→ %s", k.c.board.Titles[target])
	return nil
}

// click maneja las zonas que registra views.RenderKanban (el botón apretado).
func (k *kanbanSheet) click(z *mouse.Zone, double bool) tea.Cmd {
	switch z.Type {
	case mouse.ZoneKanbanCol:
		k.col = clamp(z.Index, 0, k.numCols()-1)
	case mouse.ZoneKanbanCard:
		parts := strings.SplitN(z.Payload, "|", 3) // columna|línea|ruta
		if len(parts) < 3 {
			return nil
		}
		var col, line int
		fmt.Sscanf(parts[0], "%d", &col)
		fmt.Sscanf(parts[1], "%d", &line)
		k.col = clamp(col, 0, k.numCols()-1)
		for i, card := range k.c.board.ColumnCards(k.col) {
			if card.NotePath == parts[2] && card.Task.Line == line {
				k.selected[k.col] = i
				k.press = &kanbanPress{col: k.col, idx: i}
			}
		}
		if double {
			k.press = nil
			return k.c.openEditor(parts[2], line)
		}
	}
	return nil
}

// motion sigue al puntero con el botón apretado: al salir de la columna de la tarjeta empieza el arrastre, y la
// columna bajo el puntero es el destino.
func (k *kanbanSheet) motion(x, width int) {
	if k.press == nil {
		return
	}
	target := k.colAt(x, width)
	if !k.drag.Active && target == k.press.col {
		return
	}
	k.drag = views.KanbanDrag{Active: true, Col: k.press.col, Idx: k.press.idx, Target: target}
}

// release termina el arrastre: si hay una tarjeta llevada a otra columna, la mueve; si se soltó donde empezó o fuera
// del tablero, no hace nada.
func (k *kanbanSheet) release() tea.Cmd {
	drag, press := k.drag, k.press
	k.drag, k.press = views.KanbanDrag{}, nil
	if press == nil || !drag.Active || drag.Target == drag.Col {
		return nil
	}
	cards := k.c.board.ColumnCards(drag.Col)
	if drag.Idx < 0 || drag.Idx >= len(cards) {
		return nil
	}
	card := cards[drag.Idx]
	return k.setColumn(&card, drag.Target)
}

// cancelDrag cancela el arrastre en curso (Esc). Devuelve si había uno.
func (k *kanbanSheet) cancelDrag() bool {
	had := k.drag.Active
	k.drag, k.press = views.KanbanDrag{}, nil
	return had
}

// colAt devuelve la columna que cae bajo la columna de pantalla x, con el reparto de ancho de RenderKanban.
func (k *kanbanSheet) colAt(x, width int) int {
	n := k.numCols()
	if n == 0 {
		return 0
	}
	return clamp(x/(width/n), 0, n-1)
}

func (k *kanbanSheet) view(r Rect, ht *mouse.HitTester) string {
	k.clampSelection()
	return views.RenderKanban(k.c.board, k.col, k.selected, r.W, r.H, ht, r.Y, k.drag)
}
