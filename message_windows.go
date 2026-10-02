//go:build windows

package main

// This file shows a Windows message box.
// The program has no console window, thus the message box is the only output.

import (
	"strings"

	"golang.org/x/sys/windows"
)

// Values for the type of the message box (MessageBoxW).
const (
	mbOK              = 0x00000000
	mbIconError       = 0x00000010
	mbIconInformation = 0x00000040
	mbSetForeground   = 0x00010000
	mbTopmost         = 0x00040000
)

// showInfo shows an information message box and waits for the user.
func showInfo(text string) {
	showMessage(text, mbIconInformation)
}

// showError shows an error message box and waits for the user.
func showError(text string) {
	showMessage(text, mbIconError)
}

// showMessage shows a message box above the browser window.
func showMessage(text string, icon uint32) {
	// A NUL character stops the text. Remove it.
	body, err := windows.UTF16PtrFromString(strings.ReplaceAll(text, "\x00", ""))
	if err != nil {
		return
	}
	caption, err := windows.UTF16PtrFromString("D-HELPER")
	if err != nil {
		return
	}
	_, _ = windows.MessageBox(0, body, caption, mbOK|icon|mbSetForeground|mbTopmost)
}
