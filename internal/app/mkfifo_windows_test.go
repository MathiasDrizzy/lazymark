//go:build windows

package app

import "errors"

// mkfifo: en Windows no hay FIFO; los tests que lo usan se saltan.
func mkfifo(string) error { return errors.New("sin FIFO en Windows") }
