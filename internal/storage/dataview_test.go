package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestReadBothDateFormats (ORD-017 F1): los dos formatos de Obsidian Tasks (emojis y Dataview, con corchetes o paréntesis, con espacios) se leen igual,
// también ⏳ (programada) y ➕ (creada); de cada campo vale la primera fecha válida; lo que no es un campo se queda en el texto.
func TestReadBothDateFormats(t *testing.T) {
	for _, c := range []struct {
		name, text string
		want       Dates
		clean      string
	}{
		{"emojis", "t 🛫 2026-05-01 📅 2026-05-10 ✅ 2026-05-09 ⏳ 2026-05-02 ➕ 2026-04-30", Dates{Start: "2026-05-01", Due: "2026-05-10", Done: "2026-05-09", Scheduled: "2026-05-02", Created: "2026-04-30"}, "t"},
		{"dataview", "t [start:: 2026-05-01] [due:: 2026-05-10] [completion:: 2026-05-09] [scheduled:: 2026-05-02] [created:: 2026-04-30]", Dates{Start: "2026-05-01", Due: "2026-05-10", Done: "2026-05-09", Scheduled: "2026-05-02", Created: "2026-04-30"}, "t"},
		{"paréntesis", "t (start:: 2026-05-01) (due:: 2026-05-10)", Dates{Start: "2026-05-01", Due: "2026-05-10"}, "t"},
		{"espacios", "t [ due :: 2026-05-10 ] [start::2026-05-01]", Dates{Start: "2026-05-01", Due: "2026-05-10"}, "t"},
		{"mezcla de formatos", "t 📅 2026-05-10 [start:: 2026-05-01]", Dates{Start: "2026-05-01", Due: "2026-05-10"}, "t"},
		{"emoji con selector", "t 📅️ 2026-05-10 ⏳️2026-05-09", Dates{Due: "2026-05-10", Scheduled: "2026-05-09"}, "t"},
		{"fecha inválida no es un campo", "t [due:: 2026-02-30] ⏳ 2026-13-01", Dates{}, "t [due:: 2026-02-30] ⏳ 2026-13-01"},
		{"primero válido", "t [due:: 2026-02-30] [due:: 2026-05-10] 📅 2026-06-01", Dates{Due: "2026-05-10"}, "t [due:: 2026-02-30]"},
		{"mayúsculas no valen", "t [Due:: 2026-05-10] [DUE:: 2026-05-11]", Dates{}, "t [Due:: 2026-05-10] [DUE:: 2026-05-11]"},
		{"corchetes que no son campos", "ver [x](http://a.b) y [nota] 2026-05-10", Dates{}, "ver [x](http://a.b) y [nota] 2026-05-10"},
		{"sin cerrar", "t [due:: 2026-05-10", Dates{}, "t [due:: 2026-05-10"},
	} {
		if got := ParseDates(c.text); got != c.want {
			t.Errorf("%s: %+v, se esperaba %+v", c.name, got, c.want)
		}
		if got := CleanTaskText(c.text); got != c.clean {
			t.Errorf("%s: texto limpio %q, se esperaba %q", c.name, got, c.clean)
		}
	}
}

// vaultWith crea una carpeta de notas con una nota con el contenido dado y devuelve su almacén (formato por defecto: sin date_format).
func vaultWith(t *testing.T, body string) (*Storage, string) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "n.md")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return New(dir), p
}

func lineOf(t *testing.T, p string, n int) string {
	t.Helper()
	b, _ := os.ReadFile(p)
	return strings.Split(string(b), "\n")[n-1]
}

// TestWriteDateFormat (ORD-017 F2): vault nuevo → Dataview; vault con tareas con emojis y ninguna con Dataview → emojis y un aviso una sola vez;
// vault con las dos → Dataview; date_format explícito manda; al editar una línea se conserva el formato que ya tenía.
func TestWriteDateFormat(t *testing.T) {
	str := func(v string) *string { return &v }
	set := func(s *Storage, p string, line int, due string) {
		t.Helper()
		if err := s.SetTaskDates(p, line, nil, str(due), time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	// vault nuevo (sin fechas): Dataview, sin aviso
	s, p := vaultWith(t, "# N\n- [ ] a\n- [ ] b\n")
	set(s, p, 2, "2026-05-10")
	if got := lineOf(t, p, 2); got != "- [ ] a [due:: 2026-05-10]" {
		t.Errorf("vault nuevo: %q", got)
	}
	if n := s.TakeDateFormatNotice(); n != "" {
		t.Errorf("sin aviso en un vault nuevo: %q", n)
	}
	// vault solo con emojis: emoji + aviso una vez
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] b\n")
	set(s, p, 3, "2026-05-10")
	if got := lineOf(t, p, 3); got != "- [ ] b 📅 2026-05-10" {
		t.Errorf("vault con emojis: %q", got)
	}
	if n := s.TakeDateFormatNotice(); !strings.Contains(n, "date_format") {
		t.Errorf("el aviso dice cómo cambiarlo: %q", n)
	}
	set(s, p, 3, "2026-05-11")
	if n := s.TakeDateFormatNotice(); n != "" {
		t.Errorf("el aviso sale una sola vez: %q", n)
	}
	// ya visto (guardado en la config): ni siquiera la primera
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] b\n")
	s.DateNoticeSeen = true
	set(s, p, 3, "2026-05-10")
	if n := s.TakeDateFormatNotice(); n != "" {
		t.Errorf("con el aviso ya visto no se repite: %q", n)
	}
	// vault con los dos formatos: Dataview
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] c [due:: 2026-01-02]\n- [ ] b\n")
	set(s, p, 4, "2026-05-10")
	if got := lineOf(t, p, 4); got != "- [ ] b [due:: 2026-05-10]" {
		t.Errorf("vault con los dos formatos: %q", got)
	}
	// date_format explícito manda sobre la excepción
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] b\n")
	s.DateFormatPref = "dataview"
	set(s, p, 3, "2026-05-10")
	if got := lineOf(t, p, 3); got != "- [ ] b [due:: 2026-05-10]" {
		t.Errorf("date_format=dataview: %q", got)
	}
	s, p = vaultWith(t, "# N\n- [ ] a [due:: 2026-01-02]\n- [ ] b\n")
	s.DateFormatPref = "emoji"
	set(s, p, 3, "2026-05-10")
	if got := lineOf(t, p, 3); got != "- [ ] b 📅 2026-05-10" {
		t.Errorf("date_format=emoji: %q", got)
	}
	// una línea conserva SU formato aunque el del vault sea otro; un campo nuevo usa el de las fechas que la línea ya tiene
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] c [due:: 2026-01-02]\n- [ ] p (due:: 2026-01-03)\n")
	s.DateFormatPref = "dataview"
	if err := s.SetTaskDates(p, 2, str("2026-02-01"), str("2026-02-05"), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if got := lineOf(t, p, 2); got != "- [ ] a 📅 2026-02-05 🛫 2026-02-01" && got != "- [ ] a 🛫 2026-02-01 📅 2026-02-05" {
		t.Errorf("la línea con emoji conserva el emoji: %q", got)
	}
	if err := s.SetTaskDates(p, 3, str("2026-02-01"), str("2026-02-05"), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if got := lineOf(t, p, 3); got != "- [ ] c [due:: 2026-02-05] [start:: 2026-02-01]" {
		t.Errorf("la línea Dataview conserva Dataview: %q", got)
	}
	if err := s.SetTaskDates(p, 4, str("2026-02-01"), nil, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if got := lineOf(t, p, 4); got != "- [ ] p (due:: 2026-01-03) (start:: 2026-02-01)" {
		t.Errorf("los paréntesis se conservan: %q", got)
	}
	// marcar como hecha: la completada va en el formato de la línea o, sin fechas, en el del vault
	s, p = vaultWith(t, "# N\n- [ ] a 📅 2026-01-01\n- [ ] b\n")
	s.DateFormatPref = "dataview"
	oldToday := Today
	Today = func() string { return "2026-10-02" }
	defer func() { Today = oldToday }()
	for _, line := range []int{2, 3} {
		if _, err := s.ToggleTask(p, line); err != nil {
			t.Fatal(err)
		}
	}
	if got := lineOf(t, p, 2); got != "- [x] a 📅 2026-01-01 ✅ 2026-10-02" {
		t.Errorf("completada en el formato de la línea (emoji): %q", got)
	}
	if got := lineOf(t, p, 3); got != "- [x] b [completion:: 2026-10-02]" {
		t.Errorf("completada en el formato del vault (Dataview): %q", got)
	}
	// desmarcar la quita en cualquiera de los dos formatos
	for _, line := range []int{2, 3} {
		if _, err := s.ToggleTask(p, line); err != nil {
			t.Fatal(err)
		}
	}
	if got := lineOf(t, p, 2); got != "- [ ] a 📅 2026-01-01" {
		t.Errorf("al desmarcar se quita el emoji: %q", got)
	}
	if got := lineOf(t, p, 3); got != "- [ ] b" {
		t.Errorf("al desmarcar se quita el campo Dataview: %q", got)
	}
}

// TestDateFormatsAgainstOracle (ORD-017): 50 líneas de tarea en los dos formatos (paréntesis, espacios, U+FE0F, fechas inválidas, repetidos, mayúsculas,
// corchetes que no son campos, \r…) con su resultado esperado, calculado por el agente de apoyo (otro modelo) solo desde las reglas. Los desacuerdos se
// revisaron uno por uno (ver ESTADO).
func TestDateFormatsAgainstOracle(t *testing.T) {
	data, err := os.ReadFile("testdata/dates-formats-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Line, Start, Due, Done, Scheduled, Created, Clean string
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 40 {
		t.Fatalf("solo %d casos", len(cases))
	}
	for _, c := range cases {
		got := ParseDates(c.Line)
		want := Dates{Start: c.Start, Due: c.Due, Done: c.Done, Scheduled: c.Scheduled, Created: c.Created}
		if got != want {
			t.Errorf("%q: fechas %+v, el oráculo dice %+v", c.Line, got, want)
		}
		if clean := CleanTaskText(strings.TrimSuffix(c.Line, "\r")); clean != c.Clean {
			t.Errorf("%q: texto limpio %q, el oráculo dice %q", c.Line, clean, c.Clean)
		}
	}
}

// TestMigrateKeepsTextAndWikilinks (ORD-017, segunda opinión): al migrar, un campo Dataview pegado a texto (`[due:: …]nota`) no se funde con él (queda un
// espacio y la fecha se sigue leyendo); y `[[due:: 2026-05-10]]` (un wikilink) no es un campo de fecha: ni se lee ni se migra.
func TestMigrateKeepsTextAndWikilinks(t *testing.T) {
	line := "- [ ] tarea [due:: 2026-05-10]nota"
	got := convertDates(line, FormatEmoji)
	if got != "- [ ] tarea 📅 2026-05-10 nota" {
		t.Errorf("pegado a texto: %q", got)
	}
	if d := ParseDates(got); d.Due != "2026-05-10" {
		t.Errorf("la fecha migrada se sigue leyendo: %+v", d)
	}
	if back := convertDates(got, FormatDataview); back != "- [ ] tarea [due:: 2026-05-10] nota" {
		t.Errorf("de vuelta: %q", back)
	}
	wl := "- [ ] ver [[due:: 2026-05-10]] y [[nota]] [due:: 2026-06-01]"
	if d := ParseDates(wl); d.Due != "2026-06-01" {
		t.Errorf("el wikilink no es un campo: %+v", d)
	}
	if conv := convertDates(wl, FormatEmoji); conv != "- [ ] ver [[due:: 2026-05-10]] y [[nota]] 📅 2026-06-01" {
		t.Errorf("el wikilink no se toca: %q", conv)
	}
}

// TestDataviewLikeTextThatIsNotAField (ORD-017 rev 2, R2-1): `[due:: 2026-05-10](https://…)` es el texto de un link Markdown y lo que está dentro de código en línea
// no es un campo: ni se lee ni se migra (el archivo no cambia en esas líneas).
func TestDataviewLikeTextThatIsNotAField(t *testing.T) {
	for _, line := range []string{
		"- [ ] Leer [due:: 2026-05-10](https://example.com)",
		"- [ ] Inline `[due:: 2026-05-11]` en código",
		"- [ ] Inline `📅 2026-05-11` con emoji",
		"- [ ] Doble ``[due:: 2026-05-12] y ` backtick`` aquí",
	} {
		if d := ParseDates(line); d != (Dates{}) {
			t.Errorf("%q: no hay campos, se leyó %+v", line, d)
		}
		if got := CleanTaskText(line); got != strings.TrimPrefix(line, "- [ ] ") && got != line {
			// el texto limpio es la línea tal cual (nada se quita)
			if !strings.Contains(got, "2026") {
				t.Errorf("%q: se quitó algo que no era un campo: %q", line, got)
			}
		}
		for _, to := range []DateFormat{FormatEmoji, FormatDataview} {
			if conv := convertDates(line, to); conv != line {
				t.Errorf("%q → %d: la línea no debía cambiar: %q", line, to, conv)
			}
		}
	}
	// fuera del código sí cuenta: una línea con las dos cosas migra solo el campo de verdad
	mixed := "- [ ] `[due:: 2026-05-11]` y [due:: 2026-05-12]"
	if conv := convertDates(mixed, FormatEmoji); conv != "- [ ] `[due:: 2026-05-11]` y 📅 2026-05-12" {
		t.Errorf("mixta: %q", conv)
	}
}

// TestRepeatedAndNonDateFields (ORD-017 rev 2, R2-2): con campos repetidos gana el primero válido y los demás no se muestran (ni en pantalla, ni en la CLI ni en el
// MCP: el texto limpio no lleva ningún [campo:: fecha] reconocido); `[due:: tomorrow]` no es una fecha (no se lee) pero cuenta como el campo `due` de la línea:
// editar lo reemplaza y nunca quedan dos `due`; editar no borra texto que no sea ese campo.
func TestRepeatedAndNonDateFields(t *testing.T) {
	line := "Dos [due:: 2026-05-01] y [due:: 2026-05-10] fin"
	if d := ParseDates(line); d.Due != "2026-05-01" {
		t.Errorf("gana el primero: %+v", d)
	}
	if got := CleanTaskText(line); got != "Dos y fin" || strings.Contains(got, "::") {
		t.Errorf("sin sintaxis en el texto limpio: %q", got)
	}
	if got := CleanTaskText("t 📅 2026-05-01 📅 2026-05-10 [start:: 2026-01-01] (start:: 2026-02-02)"); got != "t" {
		t.Errorf("también con emojis y mezclados: %q", got)
	}
	// editar el campo repetido: queda uno solo, el nuevo, y el texto de alrededor intacto
	if got := setDateIn("- [ ] "+line, DateDue, "2026-06-01", FormatDataview); got != "- [ ] Dos [due:: 2026-06-01] y fin" {
		t.Errorf("editar: %q", got)
	}
	if got := setDateIn("- [ ] "+line, DateDue, "", FormatDataview); got != "- [ ] Dos y fin" {
		t.Errorf("quitar: %q", got)
	}
	// un valor que no es fecha
	nd := "- [ ] Algo [due:: tomorrow] más"
	if d := ParseDates(nd); d.Due != "" {
		t.Errorf("no es una fecha: %+v", d)
	}
	if got := setDateIn(nd, DateDue, "2026-06-01", FormatDataview); got != "- [ ] Algo [due:: 2026-06-01] más" || strings.Count(got, "due::") != 1 {
		t.Errorf("editar reemplaza el valor que no era fecha y no deja dos due: %q", got)
	}
	if got := setDateIn(nd, DateStart, "2026-06-01", FormatDataview); got != "- [ ] Algo [due:: tomorrow] más [start:: 2026-06-01]" {
		t.Errorf("otro campo no toca el due crudo: %q", got)
	}
	if got := CleanTaskText(nd); got != nd {
		t.Errorf("lo que no es una fecha se queda como texto: %q", got)
	}
	// ida y vuelta con paréntesis: vuelve con corchetes (así escribe Obsidian Tasks) y el contenido es equivalente
	p := "- [ ] t (due:: 2026-05-10) (start:: 2026-05-01)"
	e := convertDates(p, FormatEmoji)
	back := convertDates(e, FormatDataview)
	if ParseDates(back) != ParseDates(p) || back != "- [ ] t [due:: 2026-05-10] [start:: 2026-05-01]" && back != "- [ ] t [start:: 2026-05-01] [due:: 2026-05-10]" {
		t.Errorf("ida y vuelta con paréntesis: %q → %q → %q", p, e, back)
	}
}

// TestEditRemovesRepeatedField (ORD-017 rev 2): editar un campo repetido deja uno solo (lo que documenta docs/cli.md).
func TestEditRemovesRepeatedField(t *testing.T) {
	got := setDateIn("- [ ] Dos [due:: 2026-05-01] y [due:: 2026-05-10] fin", DateDue, "2026-06-01", FormatDataview)
	if strings.Count(got, "due::") != 1 {
		t.Errorf("queda un solo due: %q", got)
	}
}

// TestReferenceLinksAndUnpairedBackticks (ORD-017 rev 3, R3-1): `[due:: …][ref]` y `[due:: …][]` son links de CommonMark (ni se leen ni se migran); un acento
// grave sin pareja es texto y no esconde el campo; con pareja (aunque sea con otro acento de por medio, como lo hace CommonMark) sí es código. Lo que no cambia:
// `[x] (url)` con espacio es campo, `\[…\]` es texto y `[…](texto)` es un link.
func TestReferenceLinksAndUnpairedBackticks(t *testing.T) {
	for _, line := range []string{"- [ ] Ref [due:: 2026-05-10][ref]", "- [ ] Ref [due:: 2026-05-10][]", `- [ ] Esc \[due:: 2026-05-12\]`, "- [ ] N [due:: 2026-05-14](urgente)"} {
		if d := ParseDates(line); d != (Dates{}) {
			t.Errorf("%q no es un campo: %+v", line, d)
		}
		if conv := convertDates(line, FormatEmoji); conv != line {
			t.Errorf("%q no se migra: %q", line, conv)
		}
	}
	if d := ParseDates("- [ ] Esp [due:: 2026-05-11] (http://x.y)"); d.Due != "2026-05-11" {
		t.Errorf("con espacio antes del paréntesis sí es campo: %+v", d)
	}
	for _, line := range []string{"- [ ] don`t y [due:: 2026-05-13] solo", "- [ ] `` y [due:: 2026-05-13]", "- [ ] ``a` [due:: 2026-05-13]"} {
		if d := ParseDates(line); d.Due != "2026-05-13" {
			t.Errorf("el acento grave sin pareja es texto: %q → %+v", line, d)
		}
	}
	// con pareja es código (CommonMark: don`t … `code` abre un code span entre el primero y el segundo acento)
	if d := ParseDates("- [ ] don`t y [due:: 2026-05-13] y `code`"); d.Due != "" {
		t.Errorf("entre dos acentos graves es código: %+v", d)
	}
}

// TestFieldFollowedByBracket (ORD-018 L12): `[due:: …][[otra nota]]` (wikilink contiguo) y `[due:: …][texto` (corchete sin cerrar) son campos: la etiqueta de un link de
// referencia debe cerrarse con `]` y no llevar `[` sin escapar. `[ref]`, `[]` y `[a\[b]` siguen siendo links; editar el campo no deja dos `due`.
func TestFieldFollowedByBracket(t *testing.T) {
	for _, line := range []string{"- [ ] Wiki [due:: 2026-05-10][[otra nota]]", "- [ ] Abierto [due:: 2026-05-11][texto", "- [ ] Abierto2 [due:: 2026-05-11][a [b]"} {
		d := ParseDates(line)
		if d.Due == "" {
			t.Errorf("%q: la fecha se lee", line)
			continue
		}
		got := setDateIn(line, DateDue, "2026-06-01", FormatDataview)
		if strings.Count(got, "due::") != 1 || !strings.Contains(got, "[due:: 2026-06-01]") {
			t.Errorf("%q: editar reemplaza el campo sin duplicar: %q", line, got)
		}
	}
	for _, line := range []string{"- [ ] A [due:: 2026-05-10][ref]", "- [ ] B [due:: 2026-05-10][]", `- [ ] C [due:: 2026-05-10][a\[b]`} {
		if d := ParseDates(line); d != (Dates{}) {
			t.Errorf("%q es un link de referencia: %+v", line, d)
		}
	}
}
