package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/MathiasDrizzy/lazymark/internal/editors"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// RunEditorPlugins implementa `lazymark editor-plugins install|uninstall [micro|vim|nano]…`:
// instala o quita los plugins que pegan imágenes desde el editor. Sin editor, actúa sobre
// los tres. Escribe en la configuración del usuario (env.Home): los tests y las pruebas
// con un HOME aislado lo apuntan a otro lado.
func RunEditorPlugins(args []string, env editors.Env, stdout, stderr io.Writer) int {
	usage := i18n.T("uso: lazymark editor-plugins install|uninstall [micro|vim|nano]…", "usage: lazymark editor-plugins install|uninstall [micro|vim|nano]…")
	if len(args) == 0 || (args[0] != "install" && args[0] != "uninstall") {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	names := args[1:]
	if len(names) == 0 {
		names = editors.Names
	}
	for _, n := range names {
		if !slices.Contains(editors.Names, n) {
			fmt.Fprintf(stderr, i18n.T("editor desconocido %q (micro, vim o nano)\n", "unknown editor %q (micro, vim or nano)\n"), n)
			return 2
		}
	}
	code := 0
	for _, n := range names {
		var done []string
		var err error
		if args[0] == "install" {
			done, err = editors.Install(n, env)
		} else {
			done, err = editors.Uninstall(n, env)
		}
		for _, line := range done {
			fmt.Fprintln(stdout, line)
		}
		switch {
		case err != nil:
			fmt.Fprintf(stderr, "%s: %v\n", n, err)
			code = 1
		case len(done) == 0:
			fmt.Fprintf(stdout, i18n.T("%s: sin cambios\n", "%s: nothing to do\n"), n)
		}
	}
	return code
}
