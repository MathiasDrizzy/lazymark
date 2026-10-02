package config

import (
	"regexp"
	"strings"
)

// KanbanColumn es una columna del tablero Kanban. ID es lo que va en el tag (`#kb/<id>`); el título visible
// puede ser libre (Title) o por idioma (Titles["es"], Titles["en"]).
type KanbanColumn struct {
	ID     string            `json:"id"`
	Title  string            `json:"title,omitempty"`
	Titles map[string]string `json:"titles,omitempty"`
}

// DefaultKanbanColumns son las columnas por defecto: todo, doing y done.
func DefaultKanbanColumns() []KanbanColumn {
	return []KanbanColumn{
		{ID: "todo", Titles: map[string]string{"es": "Por hacer", "en": "To do"}},
		{ID: "doing", Titles: map[string]string{"es": "En progreso", "en": "In progress"}},
		{ID: "done", Titles: map[string]string{"es": "Completado", "en": "Done"}},
	}
}

const (
	minKanbanColumns = 2
	maxKanbanColumns = 6
)

var kanbanIDRe = regexp.MustCompile(`^[a-z0-9_][a-z0-9_-]*$`)

// validKanbanColumns indica si cols sirve: entre 2 y 6 columnas, con ids en minúscula (letras, números, _ y -)
// y sin repetir. La última (o la que se llame "done") es la de las tareas hechas.
func validKanbanColumns(cols []KanbanColumn) bool {
	if len(cols) < minKanbanColumns || len(cols) > maxKanbanColumns {
		return false
	}
	seen := map[string]bool{}
	for _, c := range cols {
		if !kanbanIDRe.MatchString(c.ID) || seen[c.ID] {
			return false
		}
		seen[c.ID] = true
	}
	return true
}

// Title es el título visible de la columna en el idioma lang ("es" o "en"): el de ese idioma, si no el título
// libre, si no el de una columna por defecto con ese id, y si no el propio id.
func (c KanbanColumn) DisplayTitle(lang string) string {
	if t := c.Titles[lang]; t != "" {
		return t
	}
	if c.Title != "" {
		return c.Title
	}
	for _, d := range DefaultKanbanColumns() {
		if d.ID == c.ID {
			if t := d.Titles[lang]; t != "" {
				return t
			}
		}
	}
	return c.ID
}

// KanbanIDs devuelve los ids de las columnas configuradas, en orden.
func (c *Config) KanbanIDs() []string {
	ids := make([]string, len(c.KanbanColumns))
	for i, col := range c.KanbanColumns {
		ids[i] = strings.ToLower(col.ID)
	}
	return ids
}
