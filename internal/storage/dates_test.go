package storage

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func init() { Today = func() string { return "2026-10-02" } } // las pruebas no dependen del reloj

// TestParseDates (C.6): los tres campos, juntos y en cualquier orden, con o sin espacio y con el selector U+FE0F; una fecha
// que no existe en el calendario o está cortada no es un campo; de un campo repetido vale la primera fecha válida.
func TestParseDates(t *testing.T) {
	cases := []struct {
		name, text string
		want       Dates
	}{
		{"sin fechas", "tarea normal", Dates{}},
		{"vencimiento", "tarea 📅 2026-05-10", Dates{Due: "2026-05-10"}},
		{"inicio", "tarea 🛫 2026-05-01", Dates{Start: "2026-05-01"}},
		{"completada", "tarea ✅ 2026-05-09", Dates{Done: "2026-05-09"}},
		{"los tres en orden", "t 🛫 2026-05-01 📅 2026-05-10 ✅ 2026-05-09", Dates{"2026-05-01", "2026-05-10", "2026-05-09"}},
		{"los tres en otro orden", "t ✅ 2026-05-09 📅 2026-05-10 🛫 2026-05-01", Dates{"2026-05-01", "2026-05-10", "2026-05-09"}},
		{"sin espacio", "t📅2026-05-10", Dates{Due: "2026-05-10"}},
		{"con selector de variación", "t 📅️ 2026-05-10", Dates{Due: "2026-05-10"}},
		{"30 de febrero", "t 📅 2026-02-30", Dates{}},
		{"mes 13", "t 📅 2026-13-01", Dates{}},
		{"día 00", "t 📅 2026-05-00", Dates{}},
		{"bisiesto válido", "t 📅 2028-02-29", Dates{Due: "2028-02-29"}},
		{"bisiesto inválido", "t 📅 2026-02-29", Dates{}},
		{"cortada", "t 📅 2026-05-0", Dates{}},
		{"seguida de más dígitos", "t 📅 2026-05-100", Dates{}},
		{"repetida: vale la primera", "t 📅 2026-05-10 📅 2026-06-01", Dates{Due: "2026-05-10"}},
		{"repetida: primera inválida, segunda válida", "t 📅 2026-02-30 📅 2026-06-01", Dates{Due: "2026-06-01"}},
		{"emoji sin fecha", "t 📅", Dates{}},
		{"emoji con texto", "t 📅 mañana", Dates{}},
		{"con etiquetas", "t #kb/doing 📅 2026-05-10 #urgente", Dates{Due: "2026-05-10"}},
		{"otros emojis no son fechas", "🎉 fiesta 🚀 2026-05-10", Dates{}},
		{"japonés", "資料を書く 📅 2026-05-10", Dates{Due: "2026-05-10"}},
	}
	for _, c := range cases {
		if got := ParseDates(c.text); got != c.want {
			t.Errorf("%s: %+v, se esperaba %+v", c.name, got, c.want)
		}
	}
	if tasks := (&Storage{}).extractTasks("n", "/n.md", "- [ ] tarea 📅 2026-05-10 🛫 2026-05-01"); tasks[0].Dates.Due != "2026-05-10" || tasks[0].Dates.Start != "2026-05-01" {
		t.Errorf("las tareas llevan sus fechas: %+v", tasks[0].Dates)
	}
}

func TestCleanTaskTextWithDates(t *testing.T) {
	for in, want := range map[string]string{
		"tarea 📅 2026-05-10": "tarea",
		"tarea #kb/doing 🛫 2026-05-01 📅 2026-05-10 #urgente": "tarea #urgente",
		"tarea 📅 2026-02-30":              "tarea 📅 2026-02-30", // una fecha inválida se queda
		"tarea 📅 2026-05-10 📅 2026-06-01": "tarea 📅 2026-06-01", // la repetida se queda
		"tarea📅2026-05-10":                "tarea",
		"📅 2026-05-10 tarea":              "tarea",
	} {
		if got := CleanTaskText(in); got != want {
			t.Errorf("CleanTaskText(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

// TestSetDate (C.6): poner una fecha reemplaza la del campo en su sitio, o la agrega al final (antes de ✅ si lo hay, y
// antes del espacio y el \r finales); quitarla borra solo ese campo; no se toca nada más de la línea.
func TestSetDate(t *testing.T) {
	cases := []struct {
		name, line string
		f          DateField
		value      string
		want       string
	}{
		{"agregar vencimiento", "- [ ] tarea", DateDue, "2026-05-10", "- [ ] tarea 📅 2026-05-10"},
		{"agregar inicio", "- [ ] tarea", DateStart, "2026-05-01", "- [ ] tarea 🛫 2026-05-01"},
		{"reemplazar en su sitio", "- [ ] a 📅 2026-05-10 #x", DateDue, "2026-06-01", "- [ ] a 📅 2026-06-01 #x"},
		{"quitar", "- [ ] a 📅 2026-05-10 #x", DateDue, "", "- [ ] a #x"},
		{"quitar lo que no hay", "- [ ] a", DateDue, "", "- [ ] a"},
		{"antes de la completada", "- [x] a ✅ 2026-05-09", DateDue, "2026-05-10", "- [x] a 📅 2026-05-10 ✅ 2026-05-09"},
		{"la completada va al final", "- [x] a 📅 2026-05-10", DateDone, "2026-05-09", "- [x] a 📅 2026-05-10 ✅ 2026-05-09"},
		{"CRLF y espacios finales", "- [ ] a  \r", DateDue, "2026-05-10", "- [ ] a 📅 2026-05-10  \r"},
		{"quita con CRLF", "- [ ] a 📅 2026-05-10\r", DateDue, "", "- [ ] a\r"},
		{"reemplaza la primera válida", "- [ ] a 📅 2026-02-30 📅 2026-05-10 📅 2026-06-01", DateDue, "2026-07-01", "- [ ] a 📅 2026-02-30 📅 2026-07-01 📅 2026-06-01"},
		{"con selector de variación", "- [ ] a 📅️ 2026-05-10", DateDue, "2026-06-01", "- [ ] a 📅 2026-06-01"},
		{"con tag al final", "- [ ] a #kb/doing", DateDue, "2026-05-10", "- [ ] a #kb/doing 📅 2026-05-10"},
	}
	for _, c := range cases {
		if got := setDate(c.line, c.f, c.value); got != c.want {
			t.Errorf("%s: %q, se esperaba %q", c.name, got, c.want)
		}
	}
}

// TestCompletionDate (C.6): al marcar una tarea como hecha (al mover a la columna de hecho o con Espacio) se agrega
// ✅ hoy salvo que ya tenga una fecha de completada válida, y al desmarcarla o sacarla de la columna se quita.
func TestCompletionDate(t *testing.T) {
	cases := []struct {
		name, line string
		target     int
		want       string
	}{
		{"a hecho", "- [ ] a", 2, "- [x] a ✅ 2026-10-02"},
		{"a hecho conserva las demás fechas", "- [ ] a 📅 2026-05-10", 2, "- [x] a 📅 2026-05-10 ✅ 2026-10-02"},
		{"ya tenía completada", "- [ ] a ✅ 2026-01-01", 2, "- [x] a ✅ 2026-01-01"},
		{"completada inválida se reemplaza por una nueva", "- [ ] a ✅ 2026-02-30", 2, "- [x] a ✅ 2026-02-30 ✅ 2026-10-02"},
		{"de hecho a doing quita la completada", "- [x] a ✅ 2026-05-09", 1, "- [ ] a #kb/doing"},
		{"de hecho a todo quita la completada", "- [x] a 📅 2026-05-10 ✅ 2026-05-09", 0, "- [ ] a 📅 2026-05-10"},
		{"hecho a hecho no cambia", "- [x] a", 2, "- [x] a"},
		{"hecho a hecho conserva la fecha", "- [x] a ✅ 2026-05-09", 2, "- [x] a ✅ 2026-05-09"},
		{"doing a todo no agrega nada", "- [ ] a #kb/doing", 0, "- [ ] a"},
		{"al desmarcar se quitan también las completadas repetidas", "- [x] a ✅ 2026-05-01 ✅ 2026-05-02", 0, "- [ ] a"},
	}
	for _, c := range cases {
		got, err := RewriteForColumn(c.line, std, c.target)
		if err != nil || got != c.want {
			t.Errorf("%s: %q (%v), se esperaba %q", c.name, got, err, c.want)
		}
	}
}

func TestOverdue(t *testing.T) {
	for _, c := range []struct {
		done       bool
		due, today string
		want       bool
	}{
		{false, "2026-05-09", "2026-05-10", true},
		{false, "2026-05-10", "2026-05-10", false}, // vence hoy: no está vencida
		{false, "2026-05-11", "2026-05-10", false},
		{true, "2026-05-09", "2026-05-10", false}, // hecha: nunca
		{false, "", "2026-05-10", false},
		{false, "2025-12-31", "2026-01-01", true},
	} {
		if got := Overdue(c.done, c.due, c.today); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
}

// TestSetTaskDateWritesOneLine (C.6): la escritura es quirúrgica y con comprobación de mtime; las fechas inválidas y la
// de completada se rechazan sin tocar la nota.
func TestSetTaskDateWritesOneLine(t *testing.T) {
	s, p := kanbanNote(t, "# Plan\n\n- [ ] uno\n- [ ] dos 📅 2026-05-10\nfin\n")
	before, _ := os.ReadFile(p)
	if err := s.SetTaskDate(p, 3, DateDue, "2026-06-01", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTaskDate(p, 4, DateDue, "", time.Time{}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	want := strings.Replace(strings.Replace(string(before), "- [ ] uno\n", "- [ ] uno 📅 2026-06-01\n", 1), "- [ ] dos 📅 2026-05-10\n", "- [ ] dos\n", 1)
	if string(after) != want {
		t.Errorf("la nota quedó:\n%s\nse esperaba:\n%s", after, want)
	}
	snapshot, _ := os.ReadFile(p)
	for name, err := range map[string]error{
		"fecha inválida":           s.SetTaskDate(p, 3, DateDue, "2026-02-30", time.Time{}),
		"formato inválido":         s.SetTaskDate(p, 3, DateDue, "mañana", time.Time{}),
		"completada a mano":        s.SetTaskDate(p, 3, DateDone, "2026-05-09", time.Time{}),
		"línea que no es tarea":    s.SetTaskDate(p, 5, DateDue, "2026-05-09", time.Time{}),
		"línea fuera de rango":     s.SetTaskDate(p, 99, DateDue, "2026-05-09", time.Time{}),
		"nota fuera de la carpeta": s.SetTaskDate("/etc/hosts", 1, DateDue, "2026-05-09", time.Time{}),
	} {
		if err == nil {
			t.Errorf("%s: debía fallar", name)
		}
	}
	if now, _ := os.ReadFile(p); string(now) != string(snapshot) {
		t.Error("los intentos inválidos no deben tocar la nota")
	}
	// K2 también para las fechas: una nota que cambió por fuera no se pisa
	os.WriteFile(p, []byte(string(snapshot)+"agregado\n"), 0o644)
	later := time.Now().Add(5 * time.Second)
	os.Chtimes(p, later, later)
	if err := s.SetTaskDate(p, 3, DateDue, "2026-07-01", time.Now().Add(-time.Hour)); err != ErrNoteChanged {
		t.Errorf("se esperaba ErrNoteChanged: %v", err)
	}
}

// TestTaskIDIgnoresDates: el id de una tarea no cambia al ponerle, cambiarle o quitarle fechas.
func TestTaskIDIgnoresDates(t *testing.T) {
	a := taskHash(Task{Text: "comprar leche"})
	b := taskHash(Task{Text: "comprar leche 📅 2026-05-10 🛫 2026-05-01 ✅ 2026-05-09 #kb/doing"})
	if a != b {
		t.Errorf("las fechas no deben cambiar el id: %s vs %s", a, b)
	}
}

// TestDatesAgainstOracleFixtures (C.6): líneas de borde escritas por un segundo modelo (agente de apoyo) a partir de las
// reglas de las fechas, con su resultado esperado; el lector de lazymark debe coincidir en cada una. Los desacuerdos se
// revisaron a mano uno por uno (ver ESTADO).
func TestDatesAgainstOracleFixtures(t *testing.T) {
	data, err := os.ReadFile("testdata/dates-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Line      string `json:"line"`
		Start     string `json:"start"`
		Due       string `json:"due"`
		Completed string `json:"completed"`
		Clean     string `json:"clean"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 30 {
		t.Fatalf("solo %d casos", len(cases))
	}
	for _, c := range cases {
		tasks := (&Storage{}).extractTasks("n", "/n.md", c.Line)
		if len(tasks) != 1 {
			t.Errorf("%q: no se leyó como una tarea", c.Line)
			continue
		}
		got := tasks[0].Dates
		if got.Start != c.Start || got.Due != c.Due || got.Done != c.Completed {
			t.Errorf("%q: fechas %+v, el oráculo dice start=%q due=%q done=%q", c.Line, got, c.Start, c.Due, c.Completed)
		}
		if clean := CleanTaskText(tasks[0].Text); clean != c.Clean {
			t.Errorf("%q: texto limpio %q, el oráculo dice %q", c.Line, clean, c.Clean)
		}
	}
}
