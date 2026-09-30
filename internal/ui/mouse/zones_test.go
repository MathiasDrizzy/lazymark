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

func TestHitTesterLayering(t *testing.T) {
	h := NewHitTester()

	// Fondo registrado primero (menor prioridad)
	h.Register("panel-list", ZoneAction, 0, 1, 40, 20, 0, "focus-list")

	// Nota registrada después (mayor prioridad por orden inverso)
	h.Register("entry-0", ZoneNote, 0, 3, 30, 3, 0, "/notes/welcome.md")
	h.Register("entry-1", ZoneNote, 0, 4, 30, 4, 1, "/notes/work.md")

	// Clic en la fila de la nota 0 debe devolver entry-0, NO panel-list
	z, ok := h.Check(5, 3)
	if !ok || z.ID != "entry-0" || z.Type != ZoneNote {
		t.Fatalf("Esperado 'entry-0' (ZoneNote), obtenido: %+v", z)
	}

	// Clic en la fila de la nota 1 debe devolver entry-1
	z, ok = h.Check(10, 4)
	if !ok || z.ID != "entry-1" || z.Type != ZoneNote {
		t.Fatalf("Esperado 'entry-1' (ZoneNote), obtenido: %+v", z)
	}

	// Clic en espacio vacío debajo de las notas (fila 10) debe devolver panel-list
	z, ok = h.Check(5, 10)
	if !ok || z.ID != "panel-list" || z.Payload != "focus-list" {
		t.Fatalf("Esperado 'panel-list' de fondo, obtenido: %+v", z)
	}
}
