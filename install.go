package main

// This file installs and uninstalls D-HELPER for the current user.
// It does not need administrator rights.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	// installFolderName is the folder of the program in %LOCALAPPDATA%\Programs.
	installFolderName = "d-helper"
	// exeName is the name of the program file.
	exeName = "d-helper.exe"
)

// errInstalledBusy tells that the installed program is in use and D-HELPER cannot replace it.
var errInstalledBusy = errors.New("le programme installé est en cours d'utilisation. Fermez les fenêtres D-HELPER puis relancez.")

// startAction is the action of a start without arguments (double-click on the program).
type startAction int

const (
	// actionInstall copies the program and registers the protocol (no installation yet).
	actionInstall startAction = iota
	// actionUpdate replaces the installed copy and registers the protocol again.
	actionUpdate
	// actionRepair registers the protocol again. It does not copy the program.
	actionRepair
)

// chooseAction gives the action of a start without arguments.
// running is the path of the running program, installDir is the install folder,
// installed tells if the installed program exists.
func chooseAction(running, installDir string, installed bool) startAction {
	switch {
	case installed && sameFolder(filepath.Dir(running), installDir):
		return actionRepair
	case installed:
		return actionUpdate
	default:
		return actionInstall
	}
}

// sameFolder tells if two folder paths are equal.
// Windows paths are not case-sensitive, thus the comparison ignores the case.
func sameFolder(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// isInstalled tells if the installed program exists at path.
func isInstalled(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// repair registers the d-helper:// protocol again with the installed program.
// It does not copy the program. It gives the path of the installed program.
func repair() (string, error) {
	dest, err := installPath()
	if err != nil {
		return "", err
	}
	if err := registerProtocol(dest); err != nil {
		return "", err
	}
	return dest, nil
}

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
	if err := replaceFile(temp, dest); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

// replaceFile renames temp to dest. temp and dest are in the same folder.
// Windows does not let a program replace a running program file, but it lets it rename it.
// Thus, if the first rename fails, replaceFile moves dest to dest.old, then renames temp again.
// If this is not possible, it gives errInstalledBusy.
func replaceFile(temp, dest string) error {
	old := dest + ".old"
	if err := os.Rename(temp, dest); err == nil {
		// Remove the old copy of a previous update, if it exists.
		_ = os.Remove(old)
		return nil
	}
	// The old copy of a previous update can stay. Remove it first.
	_ = os.Remove(old)
	if err := os.Rename(dest, old); err != nil {
		return errInstalledBusy
	}
	if err := os.Rename(temp, dest); err != nil {
		// Put back the installed program.
		_ = os.Rename(old, dest)
		return errInstalledBusy
	}
	// This fails if the old program still runs. The next update removes it.
	_ = os.Remove(old)
	return nil
}
