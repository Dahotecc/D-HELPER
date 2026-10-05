package main

// This file finds the Dahotecc root folder on this computer.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	// envRoot is the environment variable that sets the root.
	// It has priority over the Dropbox info.json file.
	envRoot = "D_HELPER_ROOT"
	// rootFolderName is the name of the root folder in the Dropbox folder.
	rootFolderName = "Dahotecc"
)

// errRootNotFound tells that D-HELPER cannot find the Dahotecc root folder.
var errRootNotFound = errors.New("root not found")

// dropboxInfo is the part of the Dropbox info.json file that D-HELPER uses.
type dropboxInfo struct {
	Business *dropboxAccount `json:"business"`
	Personal *dropboxAccount `json:"personal"`
}

// dropboxAccount is one Dropbox account in the info.json file.
type dropboxAccount struct {
	Path string `json:"path"`
}

// findRoot gives the absolute path of the Dahotecc root folder.
// First, it uses the D_HELPER_ROOT variable.
// If this variable is empty, it reads %LOCALAPPDATA%\Dropbox\info.json.
func findRoot() (string, error) {
	if value := os.Getenv(envRoot); value != "" {
		if !filepath.IsAbs(value) {
			return "", errRootNotFound
		}
		return filepath.Clean(value), nil
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", errRootNotFound
	}
	return rootFromInfoFile(filepath.Join(localAppData, "Dropbox", "info.json"))
}

// rootFromInfoFile reads a Dropbox info.json file and gives the Dahotecc root folder.
// It uses the business account if it exists. If not, it uses the personal account.
func rootFromInfoFile(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", errRootNotFound
	}
	var info dropboxInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", errRootNotFound
	}

	var dropboxPath string
	switch {
	case info.Business != nil && info.Business.Path != "":
		dropboxPath = info.Business.Path
	case info.Personal != nil && info.Personal.Path != "":
		dropboxPath = info.Personal.Path
	default:
		return "", errRootNotFound
	}
	if !filepath.IsAbs(dropboxPath) {
		return "", errRootNotFound
	}
	return filepath.Join(dropboxPath, rootFolderName), nil
}
