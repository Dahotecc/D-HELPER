//go:build windows

package main

// This file opens a folder in the Windows explorer.

import (
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// openFolder starts explorer.exe with the absolute path of the folder.
// It does not use a shell, cmd or ShellExecute.
func openFolder(dir string) error {
	windowsDir, err := windows.GetWindowsDirectory()
	if err != nil {
		return err
	}
	explorer := filepath.Join(windowsDir, "explorer.exe")

	// A final "\" before the quote changes the meaning of the quote. Add "." after it.
	if strings.HasSuffix(dir, `\`) {
		dir += "."
	}
	// Set the full command line. Thus the path is always between quotes,
	// also when it contains a comma. The path cannot contain a quote.
	cmd := exec.Command(explorer)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: `"` + explorer + `" "` + dir + `"`,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Do not wait for explorer.exe. Its exit code is not reliable.
	return cmd.Process.Release()
}
