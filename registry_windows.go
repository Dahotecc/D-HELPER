//go:build windows

package main

// This file registers the d-helper:// protocol in the Windows registry.
// It uses HKEY_CURRENT_USER: no administrator rights are necessary.

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// protocolKey is the registry key of the protocol, in HKEY_CURRENT_USER.
const protocolKey = `Software\Classes\d-helper`

// registerProtocol tells Windows to start exe for each d-helper:// link.
// The browser gives the link as the only argument ("%1").
func registerProtocol(exe string) error {
	if err := setRegistryValue(protocolKey, "", "URL:D-HELPER"); err != nil {
		return err
	}
	// An empty "URL Protocol" value marks the key as a URL protocol.
	if err := setRegistryValue(protocolKey, "URL Protocol", ""); err != nil {
		return err
	}
	if err := setRegistryValue(protocolKey+`\DefaultIcon`, "", `"`+exe+`",0`); err != nil {
		return err
	}
	return setRegistryValue(protocolKey+`\shell\open\command`, "", `"`+exe+`" "%1"`)
}

// unregisterProtocol removes the registry key of the protocol.
// A key that does not exist is not an error.
func unregisterProtocol() error {
	// Windows deletes only keys without subkeys. Delete the subkeys first.
	keys := []string{
		protocolKey + `\shell\open\command`,
		protocolKey + `\shell\open`,
		protocolKey + `\shell`,
		protocolKey + `\DefaultIcon`,
		protocolKey,
	}
	for _, path := range keys {
		err := registry.DeleteKey(registry.CURRENT_USER, path)
		if err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
	}
	return nil
}

// setRegistryValue creates the key if necessary, then sets a string value.
// The name "" is the default value of the key.
func setRegistryValue(path, name, value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, path, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(name, value)
}
