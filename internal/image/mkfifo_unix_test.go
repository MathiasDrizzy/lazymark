//go:build !windows

package image

import "syscall"

// mkfifo crea un FIFO (los tests de archivos que no son regulares).
func mkfifo(path string) error { return syscall.Mkfifo(path, 0o644) }
