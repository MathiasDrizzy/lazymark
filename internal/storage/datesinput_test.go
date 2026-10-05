package storage

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

func day(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

// TestParseDateInput (ORD-015 C.2): lo que se escribe en el popup de fechas: absoluta, hoy, mañana, +Nd, +Nw y días de la semana; vacío quita;
// lo demás es un error. El 2026-10-05 es lunes.
func TestParseDateInput(t *testing.T) {
	for _, c := range []struct {
		today, input string
		lang         i18n.Language
		want         string
		bad          bool
	}{
		{"2026-10-05", "", i18n.LangES, "", false},
		{"2026-10-05", "   ", i18n.LangES, "", false},
		{"2026-10-05", "2026-12-31", i18n.LangES, "2026-12-31", false},
		{"2026-10-05", " 2026-12-31 ", i18n.LangEN, "2026-12-31", false},
		{"2026-10-05", "2026-02-30", i18n.LangES, "", true},
		{"2026-10-05", "2026/10/05", i18n.LangES, "", true},
		{"2026-10-05", "hoy", i18n.LangES, "2026-10-05", false},
		{"2026-10-05", "HOY", i18n.LangES, "2026-10-05", false},
		{"2026-10-05", "today", i18n.LangEN, "2026-10-05", false},
		{"2026-10-05", "mañana", i18n.LangES, "2026-10-06", false},
		{"2026-10-05", "manana", i18n.LangES, "2026-10-06", false},
		{"2026-10-05", "tomorrow", i18n.LangEN, "2026-10-06", false},
		{"2026-10-05", "+0d", i18n.LangES, "2026-10-05", false},
		{"2026-10-05", "+3d", i18n.LangES, "2026-10-08", false},
		{"2026-10-05", "+3 D", i18n.LangES, "2026-10-08", false},
		{"2026-10-05", "+1w", i18n.LangES, "2026-10-12", false},
		{"2026-10-05", "+2W", i18n.LangEN, "2026-10-19", false},
		{"2026-10-30", "+3d", i18n.LangES, "2026-11-02", false},
		{"2026-12-31", "+1d", i18n.LangES, "2027-01-01", false},
		{"2028-02-28", "+1d", i18n.LangES, "2028-02-29", false},
		{"2028-02-29", "+1d", i18n.LangES, "2028-03-01", false},
		{"2027-02-28", "+1d", i18n.LangES, "2027-03-01", false},
		{"2024-02-29", "+365d", i18n.LangES, "2025-02-28", false},
		{"2026-10-05", "+10000d", i18n.LangES, "", true},
		{"2026-10-05", "-3d", i18n.LangES, "", true},
		{"2026-10-05", "3d", i18n.LangES, "", true},
		{"2026-10-05", "+d", i18n.LangES, "", true},
		{"2026-10-05", "+1.5d", i18n.LangES, "", true},
		{"2026-10-05", "++3d", i18n.LangES, "", true},
		// días de la semana: el primero estrictamente posterior a hoy (lunes 2026-10-05)
		{"2026-10-05", "martes", i18n.LangES, "2026-10-06", false},
		{"2026-10-05", "lunes", i18n.LangES, "2026-10-12", false},
		{"2026-10-05", "miércoles", i18n.LangES, "2026-10-07", false},
		{"2026-10-05", "miercoles", i18n.LangES, "2026-10-07", false},
		{"2026-10-05", "SÁBADO", i18n.LangES, "2026-10-10", false},
		{"2026-10-05", "sabado", i18n.LangES, "2026-10-10", false},
		{"2026-10-05", "domingo", i18n.LangES, "2026-10-11", false},
		{"2026-10-05", "vie", i18n.LangES, "2026-10-09", false},
		{"2026-10-05", "friday", i18n.LangEN, "2026-10-09", false},
		{"2026-10-05", "Monday", i18n.LangEN, "2026-10-12", false},
		{"2026-10-05", "sun", i18n.LangEN, "2026-10-11", false},
		{"2026-10-05", "friday", i18n.LangES, "2026-10-09", false}, // el inglés vale siempre
		{"2026-10-05", "viernes", i18n.LangEN, "", true},           // el español solo con la interfaz en español
		{"2026-10-05", "ayer", i18n.LangES, "", true},
		{"2026-10-05", "pasado mañana", i18n.LangES, "", true},
		{"2026-10-05", "lunes martes", i18n.LangES, "", true},
		{"2026-10-05", "m", i18n.LangES, "", true}, // "mar" y "mie" son abreviaturas; una letra no
	} {
		got, err := ParseDateInput(c.input, day(c.today), c.lang)
		if c.bad {
			if err == nil {
				t.Errorf("%s %q (%s): se esperaba error, dio %q", c.today, c.input, c.lang, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s %q (%s): dio %q (%v), se esperaba %q", c.today, c.input, c.lang, got, err, c.want)
		}
	}
}

// TestWeekdayNames: el nombre del día en el idioma de la interfaz (para mostrar la fecha resuelta).
func TestWeekdayNames(t *testing.T) {
	for lang, want := range map[i18n.Language]string{i18n.LangES: "lunes", i18n.LangEN: "Monday", i18n.LangPT: "segunda-feira", i18n.LangFR: "lundi", i18n.LangDE: "Montag", i18n.LangIT: "lunedì", i18n.LangJA: "月曜日", i18n.LangZH: "星期一"} {
		if got := WeekdayName(time.Monday, lang); got != want {
			t.Errorf("%s: %q, se esperaba %q", lang, got, want)
		}
	}
}

// TestParseDateInputAgainstOracle (ORD-015 C.2): casos de borde (fin de mes, año bisiesto, +0d, días de la semana con y sin tilde, mayúsculas,
// inválidas) escritos por un segundo modelo (agente de apoyo) solo desde las reglas, con su resultado esperado calculado a mano. Los desacuerdos
// se revisaron uno por uno (ver ESTADO).
func TestParseDateInputAgainstOracle(t *testing.T) {
	data, err := os.ReadFile("testdata/dates-input-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Today  string `json:"today"`
		Lang   string `json:"lang"`
		Input  string `json:"input"`
		Expect string `json:"expect"`
		Note   string `json:"note"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 40 {
		t.Fatalf("solo %d casos", len(cases))
	}
	for _, c := range cases {
		lang := i18n.Language(c.Lang)
		got, err := ParseDateInput(c.Input, day(c.Today), lang)
		switch {
		case c.Expect == "ERROR" && err == nil:
			t.Errorf("%s %q (%s): el oráculo espera error, dio %q [%s]", c.Today, c.Input, c.Lang, got, c.Note)
		case c.Expect != "ERROR" && (err != nil || got != c.Expect):
			t.Errorf("%s %q (%s): dio %q (%v), el oráculo dice %q [%s]", c.Today, c.Input, c.Lang, got, err, c.Expect, c.Note)
		}
	}
}
