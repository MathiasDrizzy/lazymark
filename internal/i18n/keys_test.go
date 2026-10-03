package i18n

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// sourceKeys recorre el código del repo (sin tests) y devuelve cada texto traducible: inglés → español. Un texto
// traducible es un par de literales (o constantes) en una llamada a i18n.T, a una función cuyos dos primeros
// parámetros se llaman es y en, o en un Binding (campos ES y EN, o el literal posicional del keymap). Un argumento que
// no se puede resolver a un texto fijo se devuelve en dynamic: así nadie agrega un texto que se escape de los tests.
func sourceKeys(t *testing.T) (keys map[string]string, dynamic []string) {
	t.Helper()
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	var files []*ast.File
	for _, dir := range []string{"internal", "cmd"} {
		filepath.WalkDir(filepath.Join(root, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, f)
			return nil
		})
	}
	// constantes de texto de todo el repo, por nombre (basta: los nombres usados con T son únicos)
	consts := map[string]ast.Expr{}
	wrappers := map[string]bool{"T": true}
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok == token.CONST {
					for _, sp := range d.Specs {
						vs := sp.(*ast.ValueSpec)
						for i, n := range vs.Names {
							if i < len(vs.Values) {
								consts[n.Name] = vs.Values[i]
							}
						}
					}
				}
			case *ast.FuncDecl:
				var names []string
				for _, fl := range d.Type.Params.List {
					for _, n := range fl.Names {
						names = append(names, n.Name)
					}
				}
				if len(names) >= 2 && names[0] == "es" && names[1] == "en" {
					wrappers[d.Name.Name] = true
				}
			}
		}
	}
	var eval func(e ast.Expr) (string, bool)
	eval = func(e ast.Expr) (string, bool) {
		switch v := e.(type) {
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				s, err := strconv.Unquote(v.Value)
				return s, err == nil
			}
		case *ast.Ident:
			if c, ok := consts[v.Name]; ok {
				return eval(c)
			}
		case *ast.BinaryExpr:
			if v.Op == token.ADD {
				l, ok1 := eval(v.X)
				r, ok2 := eval(v.Y)
				return l + r, ok1 && ok2
			}
		case *ast.ParenExpr:
			return eval(v.X)
		}
		return "", false
	}
	keys = map[string]string{}
	passthrough := func(e ast.Expr) bool { // el cuerpo de un envoltorio reenvía sus parámetros: no es un texto
		switch v := e.(type) {
		case *ast.Ident:
			return v.Name == "es" || v.Name == "en"
		case *ast.SelectorExpr:
			return v.Sel.Name == "ES" || v.Sel.Name == "EN"
		}
		return false
	}
	add := func(pos token.Pos, es, en ast.Expr) {
		if passthrough(es) && passthrough(en) {
			return
		}
		s, ok1 := eval(es)
		e, ok2 := eval(en)
		if !ok1 || !ok2 {
			dynamic = append(dynamic, fset.Position(pos).String())
			return
		}
		if prev, dup := keys[e]; dup && prev != s {
			t.Errorf("el mismo texto en inglés %q con dos españoles distintos: %q y %q", e, prev, s)
		}
		keys[e] = s
	}
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.CallExpr:
				name := ""
				switch fn := v.Fun.(type) {
				case *ast.SelectorExpr:
					name = fn.Sel.Name
				case *ast.Ident:
					name = fn.Name
				}
				if wrappers[name] && len(v.Args) == 2 {
					if sel, ok := v.Fun.(*ast.SelectorExpr); ok && name == "T" {
						if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "i18n" {
							return true
						}
					}
					add(v.Pos(), v.Args[0], v.Args[1])
				}
			case *ast.CompositeLit:
				if at, ok := v.Type.(*ast.ArrayType); ok { // []Binding{{…}, {…}}: los elementos no repiten el tipo
					if id, ok := at.Elt.(*ast.Ident); ok && id.Name == "Binding" {
						for _, el := range v.Elts {
							if cl, ok := el.(*ast.CompositeLit); ok && cl.Type == nil && len(cl.Elts) == 6 {
								add(cl.Pos(), cl.Elts[3], cl.Elts[4])
							}
						}
					}
				}
				if id, ok := v.Type.(*ast.Ident); ok && id.Name == "Binding" {
					if len(v.Elts) == 6 { // Binding{acción, contexto, teclas, ES, EN, pie}
						add(v.Pos(), v.Elts[3], v.Elts[4])
					}
					var es, en ast.Expr
					for _, el := range v.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							switch kv.Key.(*ast.Ident).Name {
							case "ES":
								es = kv.Value
							case "EN":
								en = kv.Value
							}
						}
					}
					if es != nil && en != nil {
						add(v.Pos(), es, en)
					}
				}
			}
			return true
		})
	}
	return keys, dynamic
}

// TestDumpKeys escribe los textos traducibles (inglés → español) en el archivo de $I18N_DUMP: es la entrada de las
// traducciones (go test ./internal/i18n -run TestDumpKeys con la variable puesta).
func TestDumpKeys(t *testing.T) {
	out := os.Getenv("I18N_DUMP")
	if out == "" {
		t.Skip("sin I18N_DUMP")
	}
	keys, dynamic := sourceKeys(t)
	if len(dynamic) > 0 {
		t.Fatalf("textos que no se pueden resolver: %v", dynamic)
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	type pair struct {
		EN string `json:"en"`
		ES string `json:"es"`
	}
	var list []pair
	for _, k := range names {
		list = append(list, pair{k, keys[k]})
	}
	b, _ := json.MarshalIndent(list, "", "  ")
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d textos", len(list))
}
