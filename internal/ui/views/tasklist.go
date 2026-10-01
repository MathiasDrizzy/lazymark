package views

import (
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
	"github.com/MathiasDrizzy/lazymark/internal/storage"
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
