//go:build !windows

package main

// This file replaces path_windows.go on other systems, for the tests.

import "path/filepath"

// realPath gives the path of a file or folder without symbolic links.
func realPath(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
