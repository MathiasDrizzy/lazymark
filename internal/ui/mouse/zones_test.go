package mouse

import (
	"testing"
)

func TestHitTester(t *testing.T) {
	h := NewHitTester()

	// Registrar pestaña 1 en (0, 0) a (10, 1)
	h.Register("tab-1", ZoneTab, 0, 0, 10, 1, 0, "notas")
	// Registrar pestaña 2 en (12, 0) a (22, 1)
	h.Register("tab-2", ZoneTab, 12, 0, 22, 1, 1, "categorias")

	// Clic dentro de pestaña 1
	z, ok := h.Check(5, 0)
	if !ok || z.ID != "tab-1" || z.Index != 0 {
		t.Errorf("Hit-test falló para pestaña 1: %+v", z)
	}

	// Clic dentro de pestaña 2
	z, ok = h.Check(15, 1)
	if !ok || z.ID != "tab-2" || z.Index != 1 {
		t.Errorf("Hit-test falló para pestaña 2: %+v", z)
	}

	// Clic en espacio vacío
	_, ok = h.Check(11, 0)
	if ok {
		t.Errorf("Se esperaba fallo en zona vacía")
	}

	// Probar Clear
	h.Clear()
	_, ok = h.Check(5, 0)
	if ok {
		t.Errorf("Se esperaba fallo tras Clear")
	}
}
