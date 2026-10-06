package views

import (
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
	"sort"
)

// TaskFilter define el tipo de filtro aplicado a las tareas
type TaskFilter int

const (
	TaskFilterAll     TaskFilter = iota // Todas las tareas
	TaskFilterPending                   // Solo pendientes (- [ ])
	TaskFilterDone                      // Solo completadas (- [x])
)

// FlatTask es una tarea aplanada con contexto de la nota padre
type FlatTask struct {
	storage.Task
	NotePath string
}

// CollectTasks extrae todas las tareas de todas las notas y las aplana en una lista única
func CollectTasks(notes []storage.Note, filter TaskFilter) []FlatTask {
	var tasks []FlatTask
	for _, note := range notes {
		for _, task := range note.Tasks {
			switch filter {
			case TaskFilterPending:
				if task.Done {
					continue
				}
			case TaskFilterDone:
				if !task.Done {
					continue
				}
			}
			tasks = append(tasks, FlatTask{
				Task:     task,
				NotePath: note.Path,
			})
		}
	}
	return tasks
}

// TaskFilterLabel devuelve la etiqueta del filtro actual
func TaskFilterLabel(filter TaskFilter) string {
	switch filter {
	case TaskFilterPending:
		return i18n.T("Pendientes", "Pending")
	case TaskFilterDone:
		return i18n.T("Completadas", "Completed")
	default:
		return i18n.T("Todas", "All")
	}
}

// SortTasksByDue ordena las tareas por vencimiento (`tasks_sort = "due"`): primero las pendientes con vencimiento, de la más vencida a la más lejana (las vencidas quedan
// arriba y las próximas enseguida), después las pendientes sin vencimiento y al final las hechas; dentro de cada grupo se conserva el orden de antes.
func SortTasksByDue(tasks []FlatTask) {
	group := func(t FlatTask) int {
		switch {
		case t.Task.Done:
			return 2
		case t.Task.Dates.Due == "" || !storage.ValidDate(t.Task.Dates.Due):
			return 1
		}
		return 0
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		gi, gj := group(tasks[i]), group(tasks[j])
		if gi != gj {
			return gi < gj
		}
		return gi == 0 && tasks[i].Task.Dates.Due < tasks[j].Task.Dates.Due
	})
}
