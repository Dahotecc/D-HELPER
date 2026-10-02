//go:build !windows

package main

// This file replaces message_windows.go on other systems, for the tests.

import (
	"fmt"
	"os"
)

// showInfo writes the message on the standard error.
func showInfo(text string) {
	fmt.Fprintln(os.Stderr, text)
}

// showError writes the message on the standard error.
func showError(text string) {
	fmt.Fprintln(os.Stderr, text)
}
