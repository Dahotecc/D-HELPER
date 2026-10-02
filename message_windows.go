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
	mbYesNo           = 0x00000004
	mbIconError       = 0x00000010
	mbIconQuestion    = 0x00000020
	mbIconInformation = 0x00000040
	mbSetForeground   = 0x00010000
	mbTopmost         = 0x00040000
)

// idYes is the result of MessageBoxW when the user clicks "Oui".
const idYes = 6

// showInfo shows an information message box and waits for the user.
func showInfo(text string) {
	showMessage(text, mbOK|mbIconInformation)
}

// showError shows an error message box and waits for the user.
func showError(text string) {
	showMessage(text, mbOK|mbIconError)
}

// askYesNo shows a question with the buttons "Oui" and "Non".
// It gives true only if the user clicks "Oui".
func askYesNo(text string) bool {
	return showMessage(text, mbYesNo|mbIconQuestion) == idYes
}

// showMessage shows a message box above the browser window.
// It gives the result of MessageBoxW (the button that the user clicked), or 0 if an error occurs.
func showMessage(text string, style uint32) int32 {
	// A NUL character stops the text. Remove it.
	body, err := windows.UTF16PtrFromString(strings.ReplaceAll(text, "\x00", ""))
	if err != nil {
		return 0
	}
	caption, err := windows.UTF16PtrFromString("D-HELPER")
	if err != nil {
		return 0
	}
	result, _ := windows.MessageBox(0, body, caption, style|mbSetForeground|mbTopmost)
	return result
}
