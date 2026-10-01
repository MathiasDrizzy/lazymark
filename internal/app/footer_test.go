package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// footerLine devuelve el texto plano de la última fila (la barra de atajos).
func footerLine(m *AppModel) string {
	lines := screen(m)
	return ansi.Strip(lines[len(lines)-1])
}

// hintCount cuenta los atajos visibles de la barra: cada uno es "Acción: tecla".
func hintCount(line string) int {
	n := 0
	for _, d := range []string{"Editar:", "Nueva nota:", "Nueva carpeta:", "Renombrar:", "Mover:", "Borrar:", "Seleccionar:", "Salir:", "Atajos:"} {
		if strings.Contains(line, d) {
			n++
		}
	}
	return n
}

// TestFooterNeverEmpties (C8): la barra de atajos se recorta de a un atajo al
// achicar la terminal o alargar el mensaje de estado, pero nunca se vacía
// mientras quepa al menos uno. Antes, con ~118 columnas desaparecía entera.
func TestFooterNeverEmpties(t *testing.T) {
	statuses := []string{
		"5 notas cargadas",
		"Tarea completada: revisar el informe trimestral con el equipo de ventas",
		"La nota cambió por fuera: se recargó, vuelve a intentarlo",
		// ~105 caracteres: con el estado reservado antes que los atajos, a ~118
		// columnas no quedaba sitio ni para uno (el síntoma que halló cerebro)
		"Imagen guardada en assets/informe-trimestral-de-ventas-20261001-153045.png en la nota del equipo de ventas",
	}
	for _, status := range statuses {
		prev := -1
		for w := 200; w >= 60; w-- {
			m := newTestModel(t, w, 30)
			m.c.status = status
			line := footerLine(m)
			if got := ansi.StringWidth(line); got != w {
				t.Fatalf("w=%d: la barra mide %d celdas", w, got)
			}
			n := hintCount(line)
			if n == 0 {
				t.Fatalf("w=%d estado=%q: la barra de atajos quedó vacía: %q", w, status, line)
			}
			// monótona: al achicar, solo se pierde de a un atajo
			if prev >= 0 && (n > prev || prev-n > 1) {
				t.Errorf("w=%d estado=%q: de %d atajos pasó a %d (debe perder de a uno): %q", w, status, prev, n, line)
			}
			prev = n
		}
	}
}

// TestFooterAt118Columns (C8): el caso exacto que halló cerebro. Con ~118
// columnas y un mensaje de estado largo, la barra se vaciaba por completo.
func TestFooterAt118Columns(t *testing.T) {
	m := newTestModel(t, 118, 30)
	m.c.status = "Imagen guardada en assets/informe-trimestral-de-ventas-20261001-153045.png en la nota del equipo de ventas"
	line := footerLine(m)
	if hintCount(line) == 0 {
		t.Fatalf("a 118 columnas la barra de atajos quedó vacía: %q", line)
	}
	if !strings.Contains(line, "Atajos: ?") {
		t.Errorf("el atajo de ayuda debe conservarse siempre: %q", line)
	}
	if got := ansi.StringWidth(line); got != 118 {
		t.Errorf("la barra mide %d celdas, se esperaban 118", got)
	}
}
