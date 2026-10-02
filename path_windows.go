//go:build windows

package main

// This file gives the real path of a file or folder on Windows.

import (
	"strings"

	"golang.org/x/sys/windows"
)

const (
	// fileReadAttributes is the access right that lets D-HELPER read the attributes.
	fileReadAttributes = 0x80
	// volumeNameDOS asks for a path with a drive letter (C:\...).
	volumeNameDOS = 0x0
)

// realPath gives the final path of a file or folder.
// Windows resolves all the symbolic links and junctions in the path.
// filepath.EvalSymlinks does not resolve the junctions, thus D-HELPER does not use it on Windows.
func realPath(path string) (string, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	// FILE_FLAG_BACKUP_SEMANTICS is necessary to open a folder.
	handle, err := windows.CreateFile(name, fileReadAttributes,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, 512)
	for {
		n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), volumeNameDOS)
		if err != nil {
			return "", err
		}
		if n < uint32(len(buffer)) {
			buffer = buffer[:n]
			break
		}
		// The buffer is too small. n is the necessary size.
		buffer = make([]uint16, n)
	}

	final := windows.UTF16ToString(buffer)
	// Windows gives \\?\C:\... or \\?\UNC\server\share\... Remove this prefix.
	if strings.HasPrefix(final, `\\?\UNC\`) {
		return `\\` + strings.TrimPrefix(final, `\\?\UNC\`), nil
	}
	return strings.TrimPrefix(final, `\\?\`), nil
}
