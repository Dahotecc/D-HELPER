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

// askYesNo writes the question on the standard error and gives false ("Non").
// Thus the tests never install D-HELPER.
func askYesNo(text string) bool {
	fmt.Fprintln(os.Stderr, text)
	return false
}
