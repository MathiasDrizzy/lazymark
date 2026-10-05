//go:build windows

package safeio

import "os"

func openNonBlocking(path string) (*os.File, error) { return os.Open(path) }
