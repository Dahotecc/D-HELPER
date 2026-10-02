//go:build !windows

package main

// This file replaces explorer_windows.go on other systems, for the tests.

import "errors"

// openFolder is not available on this system.
func openFolder(dir string) error {
	return errors.New("D-HELPER ouvre les dossiers sous Windows seulement")
}
