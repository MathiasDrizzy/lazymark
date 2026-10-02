package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/MathiasDrizzy/lazymark/internal/clipboard"
	"github.com/MathiasDrizzy/lazymark/internal/i18n"
)

// RunPaste implementa `lazymark paste [--no-newline] [<nota.md>]`: guarda en assets/ junto a la nota la
// imagen que haya copiada (una captura o un archivo de imagen) e imprime en stdout solo
// su referencia markdown, `![](assets/…)`, para que un editor la inserte en el cursor.
// Sin argumento usa $LAZYMARK_NOTE, que lazymark define al abrir el editor. Si no hay
// imagen devuelve un código distinto de 0, escribe el motivo en stderr y no imprime nada.
func RunPaste(args []string, getenv func(string) string, saver *clipboard.Saver, stdout, stderr io.Writer) int {
	note, newline := "", true
	for _, a := range args {
		switch a {
		case "--no-newline", "-n": // nano inserta la salida tal cual: sin salto de línea al final
			newline = false
		default:
			if note == "" {
				note = a
			}
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
	if fi, err := os.Stat(filepath.Dir(abs)); err != nil || !fi.IsDir() {
		fmt.Fprintf(stderr, i18n.T("lazymark paste: no existe la carpeta de la nota: %s\n", "lazymark paste: the note's folder does not exist: %s\n"), filepath.Dir(abs))
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
