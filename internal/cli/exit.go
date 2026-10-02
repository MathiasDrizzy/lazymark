package cli

import "github.com/MathiasDrizzy/lazymark/internal/ops"

// Códigos de salida de la CLI (documentados en docs/cli.md); los define ops, que comparten la CLI y el MCP.
const (
	ExitOK       = ops.ExitOK
	ExitFailure  = ops.ExitFailure
	ExitUsage    = ops.ExitUsage
	ExitNotFound = ops.ExitNotFound
	ExitConflict = ops.ExitConflict
)

// ExitCode devuelve el código de salida de err (0 si es nil).
func ExitCode(err error) int { return ops.Code(err) }
