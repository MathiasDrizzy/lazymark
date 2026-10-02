package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// pasteUsage es la ayuda de `lazymark paste`.
func pasteUsage() string {
	return i18n.T(
		"uso: lazymark paste [--no-newline] [<nota.md>]\n\nGuarda la imagen copiada (una captura o un archivo de imagen) en la carpeta assets/ junto a\nla nota e imprime `![](assets/…)`. La nota debe existir y terminar en .md; sin argumento usa\n$LAZYMARK_NOTE. Opciones: -n, --no-newline (sin salto de línea final), -h, --help.\n",
		"usage: lazymark paste [--no-newline] [<note.md>]\n\nSaves the copied image (a screenshot or an image file) in the assets/ folder next to the\nnote and prints `![](assets/…)`. The note must exist and end in .md; without an argument it\nuses $LAZYMARK_NOTE. Options: -n, --no-newline (no trailing newline), -h, --help.\n")
}

// RunPaste implementa `lazymark paste [--no-newline] [<nota.md>]`: guarda en assets/ junto a la nota la
// imagen que haya copiada (una captura o un archivo de imagen) e imprime en stdout solo
// su referencia markdown, `![](assets/…)`, para que un editor la inserte en el cursor.
// Sin argumento usa $LAZYMARK_NOTE, que lazymark define al abrir el editor.
//
// Todo se valida ANTES de mirar el portapapeles: opciones conocidas, una sola nota, que exista
// y termine en .md. Si algo falla, o si no hay imagen, devuelve un código distinto de 0, escribe
// el motivo en stderr y no imprime ni escribe nada.
func RunPaste(args []string, getenv func(string) string, saver *clipboard.Saver, stdout, stderr io.Writer) int {
	note, newline := "", true
	for _, a := range args {
		switch {
		case a == "-h" || a == "--help":
			fmt.Fprint(stdout, pasteUsage())
			return 0
		case a == "--no-newline" || a == "-n": // nano inserta la salida tal cual: sin salto de línea al final
			newline = false
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, i18n.T("lazymark paste: opción desconocida %q\n\n", "lazymark paste: unknown option %q\n\n"), a)
			fmt.Fprint(stderr, pasteUsage())
			return 2
		case note != "":
			fmt.Fprintf(stderr, i18n.T("lazymark paste: sobra el argumento %q (una sola nota)\n", "lazymark paste: unexpected argument %q (one note only)\n"), a)
			return 2
		default:
			note = a
		}
	}
	if note == "" {
		note = getenv("LAZYMARK_NOTE")
	}
	if note == "" {
		fmt.Fprintln(stderr, i18n.T(
			"lazymark paste: falta la nota (pasa su ruta o define LAZYMARK_NOTE)",
			"lazymark paste: missing note (pass its path or set LAZYMARK_NOTE)"))
		return 2
	}
	abs, err := filepath.Abs(note)
	if err != nil {
		fmt.Fprintf(stderr, "lazymark paste: %v\n", err)
		return 2
	}
	if !strings.EqualFold(filepath.Ext(abs), ".md") {
		fmt.Fprintf(stderr, i18n.T("lazymark paste: %q no es una nota (debe terminar en .md)\n", "lazymark paste: %q is not a note (it must end in .md)\n"), note)
		return 2
	}
	if fi, err := os.Stat(abs); err != nil || !fi.Mode().IsRegular() {
		fmt.Fprintf(stderr, i18n.T("lazymark paste: la nota %q no existe\n", "lazymark paste: the note %q does not exist\n"), note)
		return 2
	}
	ref, err := saver.Paste(filepath.Dir(abs), filepath.Base(abs))
	if err != nil {
		fmt.Fprintf(stderr, "lazymark paste: %v\n", err)
		return 1
	}
	end := "\n"
	if !newline {
		end = ""
	}
	fmt.Fprintf(stdout, "![](%s)%s", ref, end)
	return 0
}
