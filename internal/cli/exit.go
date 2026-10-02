package cli

import (
	"errors"

	"github.com/MathiasDrizzy/lazymark/internal/storage"
)

// Códigos de salida de la CLI (documentados en docs/cli.md).
const (
	ExitOK       = 0
	ExitFailure  = 1 // el comando era válido pero falló (lectura, escritura, nota cambiada…)
	ExitUsage    = 2 // argumentos inválidos o una ruta que no es una nota de la carpeta: no se tocó nada
	ExitNotFound = 3 // la nota o la tarea pedida no existe
	ExitConflict = 4 // la nota cambió en disco mientras se escribía: no se escribió nada
)

// ExitError es un error con su código de salida.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// usageErr y los demás envuelven un error con su código.
func usageErr(err error) error { return &ExitError{Code: ExitUsage, Err: err} }

// ExitCode devuelve el código de salida de err: el de un ExitError, 2 si es una ruta fuera de la carpeta
// de notas, 4 si la nota cambió, y 1 para el resto. nil es 0.
func ExitCode(err error) int {
	var ee *ExitError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &ee):
		return ee.Code
	case errors.Is(err, storage.ErrOutsideNotes):
		return ExitUsage
	case errors.Is(err, storage.ErrNoteChanged):
		return ExitConflict
	}
	return ExitFailure
}
