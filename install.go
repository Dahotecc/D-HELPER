package main

// This file installs and uninstalls D-HELPER for the current user.
// It does not need administrator rights.

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	// installFolderName is the folder of the program in %LOCALAPPDATA%\Programs.
	installFolderName = "d-helper"
	// exeName is the name of the program file.
	exeName = "d-helper.exe"
)

// installPath gives the path of the installed program:
// %LOCALAPPDATA%\Programs\d-helper\d-helper.exe.
func installPath() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", errors.New("la variable LOCALAPPDATA est vide")
	}
	return filepath.Join(localAppData, "Programs", installFolderName, exeName), nil
}

// install copies the running program to the install path.
// Then it registers the d-helper:// protocol with this copy.
// It gives the path of the installed program.
func install() (string, error) {
	source, err := os.Executable()
	if err != nil {
		return "", err
	}
	dest, err := installPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	// Do not copy the file on itself (program started from the install path).
	if !isSameFile(source, dest) {
		if err := copyFile(source, dest); err != nil {
			return "", err
		}
	}
	if err := registerProtocol(dest); err != nil {
		return "", err
	}
	return dest, nil
}

// uninstall removes the d-helper:// protocol.
// It gives the path of the installed program, which stays on the disk.
func uninstall() (string, error) {
	if err := unregisterProtocol(); err != nil {
		return "", err
	}
	return installPath()
}

// isSameFile tells if two paths are the same file.
func isSameFile(a, b string) bool {
	infoA, err := os.Stat(a)
	if err != nil {
		return false
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(infoA, infoB)
}

// copyFile copies a file. It writes a temporary file, then renames it.
// Thus an error does not leave a half copy at the destination.
func copyFile(source, dest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	temp := dest + ".new"
	if err := os.WriteFile(temp, data, 0o755); err != nil {
		return err
	}
	if err := os.Rename(temp, dest); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
