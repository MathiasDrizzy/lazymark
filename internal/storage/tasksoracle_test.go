package storage

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// goldmarkTaskLines devuelve las líneas (desde 1) de las casillas que goldmark —el analizador de markdown que usa Glamour para
// mostrar la vista previa— reconoce como tareas.
func goldmarkTaskLines(src string) []int {
	b := []byte(src)
	doc := goldmark.New(goldmark.WithExtensions(extension.TaskList)).Parser().Parse(text.NewReader(b))
	var out []int
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*east.TaskCheckBox); ok && entering {
			off := 0
			if p := n.Parent(); p.Lines().Len() > 0 {
				off = p.Lines().At(0).Start
			}
			out = append(out, strings.Count(src[:off], "\n")+1)
		}
		return ast.WalkContinue, nil
	})
	return out
}

// TestExtractTasksAgainstGoldmark (C.3): el lector de tareas de lazymark coincide con el analizador de markdown de la vista previa
// (qué casillas son tareas y cuáles son código) en casos de listas anidadas, vallados y bloques sangrados. Lo hallaron los casos de
// la segunda opinión del agente de apoyo; el oráculo es goldmark, no mi lectura de CommonMark.
func TestExtractTasksAgainstGoldmark(t *testing.T) {
	docs := map[string]string{
		"anidadas":                    "# Plan\n- [ ] raiz\n  - [ ] hija\n    - [x] nieta\n\t- [ ] con tab\n",
		"viñetas y números":           "- [ ] a\n* [ ] b\n+ [ ] c\n1. [ ] d\n   1) [ ] e\n",
		"bajo un ítem sin casilla":    "* item\n    + [ ] bajo el item\n",
		"vallado":                     "- [ ] a\n\n```\n- [ ] código\n```\n- [ ] b\n",
		"vallado con tilde":           "~~~md\n  - [ ] código\n~~~\n- [ ] fuera\n",
		"vallado largo":               "````\n```\n- [ ] código\n```\n````\n- [ ] fuera\n",
		"vallado sin cerrar":          "- [ ] a\n```\n- [ ] x\n- [ ] y\n",
		"vallado en ítem cerrado":     "- [ ] a\n  ```\n  - [ ] dentro\n  ```\n- [ ] b\n",
		"vallado en ítem sin cerrar":  "- [ ] tarea 1\n  ```\n- [ ] tarea 2\n",
		"cierre sangrado 4 no cierra": "```\ncódigo\n    ```\n- [ ] sigue siendo código\n```\n",
		"bloque sangrado":             "Un párrafo.\n\n    - [ ] código sangrado\n",
		"párrafo tras la lista":       "- [ ] a\n\ntexto\n\n    - [ ] código\n",
		"dos blancos en una lista":    "- [ ] tarea 1\n\n\n    - [ ] sigue en la lista\n",
		"hija tras un blanco":         "- [ ] tarea 1\n\n    - [ ] hija\n",
	}
	for name, doc := range docs {
		var mine []int
		for _, tk := range (&Storage{}).extractTasks("n", "/n.md", doc) {
			mine = append(mine, tk.Line)
		}
		want := goldmarkTaskLines(doc)
		if !reflect.DeepEqual(mine, want) && !(len(mine) == 0 && len(want) == 0) {
			t.Errorf("%s:\n%q\n lazymark=%v goldmark=%v", name, doc, mine, want)
		}
	}
}
