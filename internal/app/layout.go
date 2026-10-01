package app

// Rect es un rectángulo de celdas de la pantalla.
type Rect struct{ X, Y, W, H int }

// Contains indica si la celda (x, y) cae dentro del rectángulo.
func (r Rect) Contains(x, y int) bool {
	return r.W > 0 && r.H > 0 && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Empty indica si el rectángulo no ocupa celdas (panel escondido).
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Tamaño mínimo de terminal (SPEC X6, decisión G): por debajo se muestra un aviso.
const (
	MinWidth  = 60
	MinHeight = 20
)

// Alturas mínimas al apilar la columna izquierda: por debajo, el panel se esconde.
const (
	minNotesH = 5
	minSideH  = 3
)

type panelID int

const (
	panelNotes panelID = iota
	panelTasks
	panelTags
	panelPreview
)

// Layout es la única fuente de geometría: se calcula en cada WindowSizeMsg o
// cambio de modo, y la usan tanto el render como la detección de clics.
type Layout struct {
	W, H     int
	TooSmall bool

	Notes, Tasks, Tags, Preview Rect
	Kanban                      Rect
	Footer                      Rect
	// Divider es la franja de 2 columnas (bordes de ambas columnas) que se
	// arrastra con el mouse para cambiar la proporción.
	Divider Rect
}

type layoutInput struct {
	W, H      int
	Ratio     float64
	Zoom      bool
	Focus     panelID
	TagCount  int
	ShowTasks bool
	ShowTags  bool
	Kanban    bool
}

func computeLayout(in layoutInput) Layout {
	l := Layout{W: in.W, H: in.H}
	if in.W < MinWidth || in.H < MinHeight {
		l.TooSmall = true
		return l
	}
	bodyH := in.H - 1
	l.Footer = Rect{0, bodyH, in.W, 1}
	if in.Kanban {
		l.Kanban = Rect{0, 0, in.W, bodyH}
		return l
	}
	if in.Zoom && in.Focus == panelPreview {
		l.Preview = Rect{0, 0, in.W, bodyH}
		return l
	}

	ratio := in.Ratio
	if ratio < 0.15 || ratio > 0.75 {
		ratio = 0.33
	}
	leftW := clamp(int(float64(in.W)*ratio+0.5), 20, in.W-30)
	l.Preview = Rect{leftW, 0, in.W - leftW, bodyH}
	l.Divider = Rect{leftW - 1, 0, 2, bodyH}

	hNotes, hTasks, hTags := stackHeights(bodyH, in)
	y := 0
	l.Notes = Rect{0, y, leftW, hNotes}
	y += hNotes
	if hTasks > 0 {
		l.Tasks = Rect{0, y, leftW, hTasks}
		y += hTasks
	}
	if hTags > 0 {
		l.Tags = Rect{0, y, leftW, hTags}
	}
	return l
}

// stackHeights reparte la altura de la columna izquierda. Al achicar, primero
// se esconde Categorías y después Tareas, en vez de comprimir el texto.
func stackHeights(total int, in layoutInput) (notes, tasks, tags int) {
	if in.Zoom {
		switch in.Focus {
		case panelTasks:
			if in.ShowTasks {
				return 0, total, 0
			}
		case panelTags:
			if in.ShowTags {
				return 0, 0, total
			}
		}
		return total, 0, 0
	}
	showTasks := in.ShowTasks && total >= minNotesH+minSideH
	showTags := in.ShowTags && total >= minNotesH+minSideH*2
	if !showTasks && in.ShowTags && total >= minNotesH+minSideH {
		showTags = true
	}

	if showTags {
		// Categorías es compacto: sus tags + 2 filas de borde, hasta un cuarto.
		tags = clamp(in.TagCount+2, minSideH, max(minSideH, total/4))
	}
	rest := total - tags
	if showTasks {
		tasks = clamp(rest*35/100, minSideH, rest-minNotesH)
	}
	return rest - tasks, tasks, tags
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
