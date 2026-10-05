package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazymark/internal/config"
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
	case actMoveCardUp:
		k.reorder(-1)
	case actMoveCardDown:
		k.reorder(1)
	case actToggleTask:
		if card := k.current(); card != nil {
			target := k.c.cols().DoneIndex()
			if card.Column == target {
				target = 0
			}
			return k.setColumn(card, target)
		}
	case actDates:
		if card := k.current(); card != nil {
			k.openDates(card)
		}
	case actEdit:
		if card := k.current(); card != nil {
			return k.c.openEditor(card.NotePath, card.Task.Line)
		}
	}
	return nil
}

// openDates abre el popup de fechas de la tarjeta; al guardar el cursor sigue a la tarjeta (la nota pasa a ser la más reciente y se reordena).
func (k *kanbanSheet) openDates(card *views.KanbanCard) {
	path, line := card.NotePath, card.Task.Line
	k.c.openDates(path, line, card.Task.Text, card.Task.Dates, func() {
		k.c.reload()
		k.clampSelection()
		for c := range k.selected {
			for i, cd := range k.c.board.ColumnCards(c) {
				if cd.NotePath == path && cd.Task.Line == line {
					k.col, k.selected[c] = c, i
				}
			}
		}
	})
}

func (k *kanbanSheet) moveTo(target int) tea.Cmd {
	if card := k.current(); card != nil && target >= 0 && target < k.numCols() {
		return k.setColumn(card, target)
	}
	return nil
}

// reorder sube (dir -1) o baja (dir 1) un lugar la tarjeta seleccionada dentro de su columna: intercambia su tarea, con todo lo que le
// pertenece (subtareas, párrafos, bloques de código), con la de la hermana vecina; las tarjetas de por medio que son subtareas (de la
// propia tarea al bajar, de la hermana anterior al subir) se saltan. Solo se reordena dentro de una nota y entre tareas hermanas: entre
// notas distintas el orden de las tarjetas es el de las notas (docs/configuration.md). Devuelve si se movió.
func (k *kanbanSheet) reorder(dir int) bool {
	cards := k.cards()
	i := clamp(k.selection(k.col), 0, max(0, len(cards)-1))
	if len(cards) == 0 || i+dir < 0 || i+dir >= len(cards) {
		return false
	}
	a := cards[i]
	var b views.KanbanCard
	found := false
	for n := i + dir; n >= 0 && n < len(cards); n += dir {
		c := cards[n]
		if c.NotePath != a.NotePath {
			k.c.setStatus("%s", i18n.T("Entre notas: sigue su orden", "Across notes: note order"))
			return false
		}
		if c.Task.Indent > a.Task.Indent {
			continue
		}
		b, found = c, true
		break
	}
	if !found {
		return false
	}
	newA, _, err := k.c.store.SwapTasks(a.NotePath, a.Task.Line, b.Task.Line, k.noteTime(a.NotePath))
	switch {
	case errors.Is(err, storage.ErrNoteChanged):
		k.c.reload()
		k.clampSelection()
		k.c.setStatus("%s", i18n.T("Cambió por fuera: recargada", "Changed outside: reloaded"))
		return false
	case errors.Is(err, storage.ErrNotSiblings):
		k.c.setStatus("%s", i18n.T("Solo entre tareas hermanas", "Only between sibling tasks"))
		return false
	case err != nil:
		k.c.errStatus("No se pudo reordenar la tarjeta", "Could not reorder card", err)
		return false
	}
	k.c.reload()
	k.clampSelection()
	// el cursor sigue a la tarjeta movida (la misma nota, en su línea nueva)
	for n, c := range k.c.board.ColumnCards(k.col) {
		if c.NotePath == a.NotePath && c.Task.Line == newA {
			k.selected[k.col] = n
			break
		}
	}
	return true
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
func (k *kanbanSheet) click(z *mouse.Zone, y int, double bool) tea.Cmd {
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
		// un clic en la fila de fechas de la tarjeta (la última antes del borde de abajo) abre el popup de fechas
		if card := k.current(); card != nil && card.NotePath == parts[2] && card.Task.Line == line && card.Task.Dates != (storage.Dates{}) && y == z.Y2-1 {
			k.press = nil
			k.openDates(card)
			return nil
		}
		if double {
			k.press = nil
			return k.c.openEditor(parts[2], line)
		}
	}
	return nil
}

// motion sigue al puntero con el botón apretado: al salir de la columna de la tarjeta empieza el arrastre, y la
// columna bajo el puntero es el destino. Dentro de la misma columna, pasar sobre otra tarjeta es reordenar: esa tarjeta es el destino.
func (k *kanbanSheet) motion(x, y, width int, ht *mouse.HitTester) {
	if k.press == nil {
		return
	}
	target := k.colAt(x, width)
	idx := -1
	if target == k.press.col && ht != nil {
		if z, ok := ht.Check(x, y); ok && z.Type == mouse.ZoneKanbanCard && strings.HasPrefix(z.Payload, fmt.Sprintf("%d|", target)) {
			idx = z.Index
		}
	}
	if !k.drag.Active && target == k.press.col && (idx < 0 || idx == k.press.idx) {
		return
	}
	k.drag = views.KanbanDrag{Active: true, Col: k.press.col, Idx: k.press.idx, Target: target, TargetIdx: idx}
}

// release termina el arrastre: si hay una tarjeta llevada a otra columna, la mueve; si se soltó sobre otra tarjeta de su columna,
// la reordena (un lugar a la vez, hasta donde se pueda); si se soltó donde empezó o fuera del tablero, no hace nada.
func (k *kanbanSheet) release() tea.Cmd {
	drag, press := k.drag, k.press
	k.drag, k.press = views.KanbanDrag{}, nil
	if press == nil || !drag.Active {
		return nil
	}
	cards := k.c.board.ColumnCards(drag.Col)
	if drag.Idx < 0 || drag.Idx >= len(cards) {
		return nil
	}
	if drag.Target == drag.Col {
		if drag.TargetIdx < 0 || drag.TargetIdx == drag.Idx {
			return nil
		}
		k.col = drag.Col
		k.selected[k.col] = drag.Idx
		dir := 1
		if drag.TargetIdx < drag.Idx {
			dir = -1
		}
		for n := abs(drag.TargetIdx - drag.Idx); n > 0 && k.reorder(dir); n-- {
		}
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
	return views.RenderKanban(k.c.board, k.col, k.selected, r.W, r.H, ht, r.Y, k.drag, views.KanbanOptions{Cards: k.c.cfg.KanbanCards != config.KanbanCardsCompact, Today: storage.Today()})
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
