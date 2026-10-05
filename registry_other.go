//go:build !windows

package main

// This file replaces registry_windows.go on other systems, for the tests.

import "errors"

// errWindowsOnly tells that the registry is available on Windows only.
var errWindowsOnly = errors.New("D-HELPER s'installe sous Windows seulement")

// registerProtocol is not available on this system.
func registerProtocol(exe string) error {
	return errWindowsOnly
}

// unregisterProtocol is not available on this system.
func unregisterProtocol() error {
	return errWindowsOnly
}
