package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"github.com/MathiasDrizzy/lazymark/internal/ui/mouse"
	"github.com/MathiasDrizzy/lazymark/internal/ui/views"
)

var stages = [3]storage.TaskStage{storage.StageTodo, storage.StageDoing, storage.StageDone}

// kanbanSheet es la Hoja 2: tablero de tres columnas con las tareas.
type kanbanSheet struct {
	c        *core
	col      int
	selected [3]int
}

func (k *kanbanSheet) cards() []views.KanbanCard { return k.c.board.ColumnCards(k.col) }

func (k *kanbanSheet) current() *views.KanbanCard {
	cards := k.cards()
	if len(cards) == 0 {
		return nil
	}
	i := clamp(k.selected[k.col], 0, len(cards)-1)
	return &cards[i]
}

// clampSelection mantiene cada cursor de columna dentro de sus tarjetas.
func (k *kanbanSheet) clampSelection() {
	for c := range stages {
		k.selected[c] = clamp(k.selected[c], 0, max(0, len(k.c.board.ColumnCards(c))-1))
	}
}

func (k *kanbanSheet) key(a Action) tea.Cmd {
	switch a {
	case actLeft:
		k.col = (k.col + 2) % 3
	case actRight, actNextPanel:
		k.col = (k.col + 1) % 3
	case actPanelNotes, actPanelTasks, actPanelTags:
		k.col = int(a - actPanelNotes)
	case actUp:
		k.selected[k.col] = max(0, k.selected[k.col]-1)
	case actDown:
		k.selected[k.col] = min(max(0, len(k.cards())-1), k.selected[k.col]+1)
	case actMoveCardLeft:
		k.moveTo(k.col - 1)
	case actMoveCardRight:
		k.moveTo(k.col + 1)
	case actToggleTask:
		if card := k.current(); card != nil {
			target := 2
			if card.Stage == storage.StageDone {
				target = 0
			}
			k.setStage(card, target)
		}
	case actEdit:
		if card := k.current(); card != nil {
			return k.c.openEditor(card.NotePath, card.Task.Line)
		}
	}
	return nil
}

func (k *kanbanSheet) moveTo(target int) {
	if card := k.current(); card != nil && target >= 0 && target <= 2 {
		k.setStage(card, target)
	}
}

// setStage reescribe solo la línea de la tarea y deja el foco en la columna destino.
func (k *kanbanSheet) setStage(card *views.KanbanCard, target int) {
	if err := k.c.store.UpdateTaskStage(card.NotePath, card.Task.Line, stages[target]); err != nil {
		k.c.errStatus("No se pudo mover la tarjeta", "Could not move card", err)
		return
	}
	k.c.reload()
	k.col = target
	// el cursor sigue a la tarjeta movida (la misma nota y línea), no a la última de la columna
	k.selected[target] = max(0, len(k.c.board.ColumnCards(target))-1)
	for i, c := range k.c.board.ColumnCards(target) {
		if c.NotePath == card.NotePath && c.Task.Line == card.Task.Line {
			k.selected[target] = i
			break
		}
	}
	k.clampSelection()
	names := []string{i18n.T("Por hacer", "To do"), i18n.T("En progreso", "In progress"), i18n.T("Completado", "Done")}
	k.c.setStatus("→ %s", names[target])
}

// click maneja las zonas que registra views.RenderKanban.
func (k *kanbanSheet) click(z *mouse.Zone, double bool) tea.Cmd {
	switch z.Type {
	case mouse.ZoneKanbanCol:
		k.col = z.Index
	case mouse.ZoneKanbanCard:
		parts := strings.SplitN(z.Payload, ":", 3)
		if len(parts) < 3 {
			return nil
		}
		var col, line int
		fmt.Sscanf(parts[0], "%d", &col)
		fmt.Sscanf(parts[2], "%d", &line)
		k.col = clamp(col, 0, 2)
		for i, card := range k.c.board.ColumnCards(k.col) {
			if card.NotePath == parts[1] && card.Task.Line == line {
				k.selected[k.col] = i
			}
		}
		if double {
			return k.c.openEditor(parts[1], line)
		}
	}
	return nil
}

func (k *kanbanSheet) view(r Rect, ht *mouse.HitTester) string {
	k.clampSelection()
	return views.RenderKanban(k.c.board, k.col, k.selected, r.W, r.H, ht, r.Y)
}
