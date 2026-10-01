package mouse

// ZoneType define el tipo de elemento clickeable
type ZoneType string

const (
	ZoneTab     ZoneType = "tab"
	ZoneNote    ZoneType = "note"
	ZoneAction  ZoneType = "action"
	ZonePreview ZoneType = "preview"
	ZoneTag        ZoneType = "tag"
	ZoneTask       ZoneType = "task"
	ZoneGallery    ZoneType = "gallery"
	ZoneKanbanCol  ZoneType = "kanban_col"
	ZoneKanbanCard ZoneType = "kanban_card"
)

// Zone representa una región rectangular en la pantalla de la terminal
type Zone struct {
	ID       string
	Type     ZoneType
	X1       int
	Y1       int
	X2       int
	Y2       int
	Index    int
	Payload  string
}

// HitTester gestiona el registro y verificación de coordenadas de clic
type HitTester struct {
	zones []Zone
}

func NewHitTester() *HitTester {
	return &HitTester{zones: make([]Zone, 0)}
}

// Clear vacía las zonas registradas (se llama en cada ciclo de renderizado)
func (h *HitTester) Clear() {
	h.zones = h.zones[:0]
}

// Register añade una nueva zona clickeable
func (h *HitTester) Register(id string, zType ZoneType, x1, y1, x2, y2 int, index int, payload string) {
	h.zones = append(h.zones, Zone{
		ID:      id,
		Type:    zType,
		X1:      x1,
		Y1:      y1,
		X2:      x2,
		Y2:      y2,
		Index:   index,
		Payload: payload,
	})
}

// Check determina si las coordenadas (x, y) caen dentro de alguna zona
func (h *HitTester) Check(x, y int) (*Zone, bool) {
	// Evaluar en orden inverso (elementos superiores primero)
	for i := len(h.zones) - 1; i >= 0; i-- {
		z := &h.zones[i]
		if x >= z.X1 && x <= z.X2 && y >= z.Y1 && y <= z.Y2 {
			return z, true
		}
	}
	return nil, false
}

// Zones devuelve una copia de las zonas registradas (útil para pruebas y depuración)
func (h *HitTester) Zones() []Zone {
	result := make([]Zone, len(h.zones))
	copy(result, h.zones)
	return result
}
