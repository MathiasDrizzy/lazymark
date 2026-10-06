package storage

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// ErrBadDateInput es el error de ParseDateInput cuando el texto no es una fecha que se entienda.
var ErrBadDateInput = i18n.NewError("no entiendo esa fecha", "not a date I understand")

// weekdayNames son los nombres de los días de la semana en cada idioma de la interfaz, de domingo (0) a sábado (6), con sus variantes.
// El primero de cada día es el que se muestra.
var weekdayNames = map[i18n.Language][7][]string{
	i18n.LangEN: {{"Sunday"}, {"Monday"}, {"Tuesday"}, {"Wednesday"}, {"Thursday"}, {"Friday"}, {"Saturday"}},
	i18n.LangES: {{"domingo"}, {"lunes"}, {"martes"}, {"miércoles"}, {"jueves"}, {"viernes"}, {"sábado"}},
	i18n.LangPT: {{"domingo"}, {"segunda-feira", "segunda"}, {"terça-feira", "terça"}, {"quarta-feira", "quarta"}, {"quinta-feira", "quinta"}, {"sexta-feira", "sexta"}, {"sábado"}},
	i18n.LangFR: {{"dimanche"}, {"lundi"}, {"mardi"}, {"mercredi"}, {"jeudi"}, {"vendredi"}, {"samedi"}},
	i18n.LangDE: {{"Sonntag"}, {"Montag"}, {"Dienstag"}, {"Mittwoch"}, {"Donnerstag"}, {"Freitag"}, {"Samstag", "Sonnabend"}},
	i18n.LangIT: {{"domenica"}, {"lunedì"}, {"martedì"}, {"mercoledì"}, {"giovedì"}, {"venerdì"}, {"sabato"}},
	i18n.LangJA: {{"日曜日", "日曜"}, {"月曜日", "月曜"}, {"火曜日", "火曜"}, {"水曜日", "水曜"}, {"木曜日", "木曜"}, {"金曜日", "金曜"}, {"土曜日", "土曜"}},
	i18n.LangZH: {{"星期日", "星期天", "周日", "周天"}, {"星期一", "周一"}, {"星期二", "周二"}, {"星期三", "周三"}, {"星期四", "周四"}, {"星期五", "周五"}, {"星期六", "周六"}},
}

// todayWords y tomorrowWords son "hoy" y "mañana" en cada idioma (sin tildes, en minúscula).
var (
	todayWords    = map[i18n.Language][]string{i18n.LangEN: {"today"}, i18n.LangES: {"hoy"}, i18n.LangPT: {"hoje"}, i18n.LangFR: {"aujourdhui"}, i18n.LangDE: {"heute"}, i18n.LangIT: {"oggi"}, i18n.LangJA: {"今日"}, i18n.LangZH: {"今天"}}
	tomorrowWords = map[i18n.Language][]string{i18n.LangEN: {"tomorrow"}, i18n.LangES: {"manana"}, i18n.LangPT: {"amanha"}, i18n.LangFR: {"demain"}, i18n.LangDE: {"morgen"}, i18n.LangIT: {"domani"}, i18n.LangJA: {"明日"}, i18n.LangZH: {"明天"}}
)

// WeekdayName es el nombre del día de la semana en el idioma lang (el inglés si no hay tabla).
func WeekdayName(d time.Weekday, lang i18n.Language) string {
	names, ok := weekdayNames[lang]
	if !ok {
		names = weekdayNames[i18n.LangEN]
	}
	return names[d][0]
}

// foldAccents pone en minúscula y quita tildes y signos (también el apóstrofo de "aujourd'hui"), para comparar sin ellos.
var accentFold = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "í", "i", "ì", "i", "î", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o", "ú", "u", "ù", "u", "û", "u", "ü", "u", "ñ", "n", "ç", "c", "ß", "ss", "'", "", "’", "")

func fold(s string) string { return accentFold.Replace(strings.ToLower(strings.TrimSpace(s))) }

var relativeRe = regexp.MustCompile(`^\+(\d{1,4}) ?([dw])$`)

// ParseDateInput resuelve lo que se escribe en el campo de una fecha a "AAAA-MM-DD" respecto de today:
//   - vacío → "" (quitar la fecha);
//   - una fecha AAAA-MM-DD que exista;
//   - hoy y mañana (en el idioma de la interfaz o en inglés);
//   - +Nd y +Nw (N de 0 a 9999): hoy más N días o semanas;
//   - un día de la semana (en el idioma de la interfaz o en inglés, con o sin tilde, o su abreviatura de 3 letras si es única):
//     el primero estrictamente posterior a hoy (si hoy es viernes, "viernes" es dentro de 7 días).
//
// Mayúsculas y espacios de los lados no importan. Lo demás es ErrBadDateInput.
func ParseDateInput(input string, today time.Time, lang i18n.Language) (string, error) {
	in := strings.TrimSpace(input)
	if in == "" {
		return "", nil
	}
	if ValidDate(in) {
		return in, nil
	}
	today = time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, time.UTC) // sin sorpresas de horario de verano
	key := fold(in)
	langs := []i18n.Language{lang}
	if lang != i18n.LangEN {
		langs = append(langs, i18n.LangEN) // el inglés vale siempre
	}
	for _, l := range langs {
		for _, w := range todayWords[l] {
			if key == w {
				return today.Format("2006-01-02"), nil
			}
		}
		for _, w := range tomorrowWords[l] {
			if key == w {
				return today.AddDate(0, 0, 1).Format("2006-01-02"), nil
			}
		}
	}
	if m := relativeRe.FindStringSubmatch(key); m != nil {
		n, _ := strconv.Atoi(m[1])
		if m[2] == "w" {
			n *= 7
		}
		d := today.AddDate(0, 0, n)
		if d.Year() > 9999 {
			return "", fmt.Errorf("%w: %q", ErrBadDateInput, in)
		}
		return d.Format("2006-01-02"), nil
	}
	for _, l := range langs {
		if wd, ok := matchWeekday(key, l); ok {
			delta := (int(wd) - int(today.Weekday()) + 7) % 7
			if delta == 0 {
				delta = 7
			}
			return today.AddDate(0, 0, delta).Format("2006-01-02"), nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrBadDateInput, in)
}

// matchWeekday reconoce key (ya sin tildes ni mayúsculas) como un día de la semana del idioma l: su nombre, una variante o una abreviatura de
// 3 letras que no sea ambigua.
func matchWeekday(key string, l i18n.Language) (time.Weekday, bool) {
	names, ok := weekdayNames[l]
	if !ok {
		return 0, false
	}
	for d, variants := range names {
		for _, v := range variants {
			if fold(v) == key {
				return time.Weekday(d), true
			}
		}
	}
	if len([]rune(key)) == 3 && key[0] < 0x80 { // abreviatura de 3 letras (solo alfabetos latinos)
		found, n := time.Sunday, 0
		for d, variants := range names {
			if strings.HasPrefix(fold(variants[0]), key) {
				found, n = time.Weekday(d), n+1
			}
		}
		if n == 1 {
			return found, true
		}
	}
	return 0, false
}
