package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
)

var (
	verbRe = regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)
	codeRe = regexp.MustCompile("`[^`]*`")
	flagRe = regexp.MustCompile(`--[a-z][a-z-]*|\$[A-Z_]+`)
	keyRe  = regexp.MustCompile(`\((?:[A-Z]|Esc|Enter|Space|Tab|\?|,|[Cc]trl\+[A-Za-z])\)`) // un atajo entre paréntesis: (W), (Esc), (Space)
)

func readCatalog(t *testing.T, l Language) map[string]string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("catalog", string(l)+".json"))
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("%s.json: %v", l, err)
	}
	return m
}

func sorted(ss []string) string {
	sort.Strings(ss)
	return strings.Join(ss, "|")
}

// TestCatalogsCoverEverySourceText (C.5): ninguna clave queda sin traducir en ningún idioma, y los catálogos no
// guardan textos que ya no existen. Los textos salen del código con go/ast (keys_test.go).
func TestCatalogsCoverEverySourceText(t *testing.T) {
	keys, dynamic := sourceKeys(t)
	if len(dynamic) > 0 {
		t.Fatalf("hay textos de interfaz que no son fijos y se escapan de la cobertura: %v", dynamic)
	}
	if len(keys) < 100 {
		t.Fatalf("solo se encontraron %d textos: el extractor no está leyendo el código", len(keys))
	}
	for _, l := range Languages {
		if l == LangEN || l == LangES {
			continue
		}
		cat := readCatalog(t, l)
		missing := 0
		for en := range keys {
			if strings.TrimSpace(cat[en]) == "" {
				missing++
				if missing <= 5 {
					t.Errorf("%s: sin traducir %q", l, en)
				}
			}
		}
		if missing > 5 {
			t.Errorf("%s: %d textos sin traducir en total", l, missing)
		}
		for en := range cat {
			if _, ok := keys[en]; !ok {
				t.Errorf("%s: el catálogo guarda un texto que ya no existe en el código: %q", l, en)
			}
		}
	}
}

// TestCatalogsKeepTheLiteralParts (C.5): una traducción conserva lo que no se traduce: los marcadores de formato, el
// código entre comillas inversas, las opciones de la línea de comandos (--json), las variables ($LAZYMARK_NOTE), los
// atajos entre paréntesis ((W), (Esc)) y los saltos de línea.
func TestCatalogsKeepTheLiteralParts(t *testing.T) {
	keys, _ := sourceKeys(t)
	for _, l := range Languages {
		if l == LangEN || l == LangES {
			continue
		}
		for en, tr := range readCatalog(t, l) {
			if _, ok := keys[en]; !ok || tr == "" {
				continue
			}
			for name, re := range map[string]*regexp.Regexp{"marcadores": verbRe, "código": codeRe, "opciones": flagRe, "atajos": keyRe} {
				if a, b := sorted(re.FindAllString(en, -1)), sorted(re.FindAllString(tr, -1)); a != b {
					t.Errorf("%s: %s distintos entre %q y su traducción %q (%q vs %q)", l, name, en, tr, a, b)
				}
			}
			if strings.Count(en, "\n") != strings.Count(tr, "\n") {
				t.Errorf("%s: distinto número de saltos de línea en %q", l, en)
			}
			if strings.HasPrefix(en, " ") != strings.HasPrefix(tr, " ") || strings.HasSuffix(en, " ") != strings.HasSuffix(tr, " ") {
				t.Errorf("%s: espacios al borde distintos en %q", l, en)
			}
		}
	}
}

func TestParseAndDetect(t *testing.T) {
	for in, want := range map[string]Language{
		"en": LangEN, "EN": LangEN, "es": LangES, "pt": LangPT, "pt-BR": LangPT, "pt_BR.UTF-8": LangPT, "fr_FR.UTF-8": LangFR,
		"de": LangDE, "it": LangIT, "ja": LangJA, "ja_JP.UTF-8": LangJA, "zh-CN": LangZH, "zh_CN.UTF-8": LangZH, "zh": LangZH,
		"español": LangES, "日本語": LangJA, "简体中文": LangZH,
	} {
		if got, ok := Parse(in); !ok || got != want {
			t.Errorf("Parse(%q) = %q, %v; se esperaba %q", in, got, ok, want)
		}
	}
	if _, ok := Parse("klingon"); ok {
		t.Error("un idioma desconocido no se reconoce")
	}
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for name, c := range map[string]struct {
		env  map[string]string
		want Language
	}{
		"LANG pt_BR":              {map[string]string{"LANG": "pt_BR.UTF-8"}, LangPT},
		"LC_ALL gana a LANG":      {map[string]string{"LC_ALL": "de_DE.UTF-8", "LANG": "es_ES.UTF-8"}, LangDE},
		"LC_MESSAGES gana a LANG": {map[string]string{"LC_MESSAGES": "it_IT.UTF-8", "LANG": "es_ES.UTF-8"}, LangIT},
		"ja":                      {map[string]string{"LANG": "ja_JP.UTF-8"}, LangJA},
		"zh":                      {map[string]string{"LANG": "zh_CN.UTF-8"}, LangZH},
		"C es inglés":             {map[string]string{"LANG": "C"}, LangEN},
		"sin variables":           {map[string]string{}, LangEN},
		"idioma no soportado":     {map[string]string{"LANG": "ru_RU.UTF-8"}, LangEN},
		"es":                      {map[string]string{"LANG": "es_AR.UTF-8"}, LangES},
		"C.UTF-8 no elige idioma": {map[string]string{"LC_ALL": "C.UTF-8", "LANG": "es_ES.UTF-8"}, LangES},
		"POSIX no elige idioma":   {map[string]string{"LC_ALL": "POSIX", "LC_MESSAGES": "fr_FR.UTF-8"}, LangFR},
	} {
		if got := Detect(env(c.env)); got != c.want {
			t.Errorf("%s: %q, se esperaba %q", name, got, c.want)
		}
	}
	// "es" dentro de otra palabra ya no cuenta (antes "LANG=fr_FR.UTF-8 LC_MESSAGES=…es…" podía confundirse)
	if got := Detect(env(map[string]string{"LANG": "en_US.UTF-8", "LC_MESSAGES": "", "LC_ALL": ""})); got != LangEN {
		t.Errorf("en_US debe ser inglés: %q", got)
	}
}

func TestTranslatesFromCatalog(t *testing.T) {
	defer SetLanguage("es")
	catalogs = nil
	catalogOnce = sync.Once{}
	SetLanguage("es")
	if T("Hola", "Hello") != "Hola" {
		t.Error("es")
	}
	SetLanguage("zh")
	if CurrentLanguage() != LangZH || T("Hola", "texto-que-no-esta-en-el-catalogo") != "texto-que-no-esta-en-el-catalogo" {
		t.Error("un texto sin traducción cae en inglés")
	}
	SetLanguage("auto")
	if CurrentLanguage() != Detect(os.Getenv) {
		t.Error("auto = el idioma del sistema")
	}
	SetLanguage("pt-BR")
	if CurrentLanguage() != LangPT {
		t.Error("pt-BR")
	}
	for _, l := range Languages {
		if l.Name() == "" {
			t.Errorf("%s sin nombre", l)
		}
		if got, ok := Parse(l.Name()); !ok || got != l { // el nombre que muestra Ajustes también se entiende
			t.Errorf("Parse(%q) = %q, %v; se esperaba %q", l.Name(), got, ok, l)
		}
	}
}
