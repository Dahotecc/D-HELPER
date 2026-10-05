package main

// This file reads a d-helper:// link and makes sure that it is safe.
// Any web site can start a d-helper:// link. Do not trust the link.
// The security rules come from the D-HELPER contract (refer to README.md).

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// linkScheme is the URL scheme of the links.
	linkScheme = "d-helper"
	// linkActionOpen is the only action. It is the host part of the URL.
	linkActionOpen = "open"
	// linkParamPath is the only parameter. It is the relative path of the folder.
	linkParamPath = "path"
	// maxLinkLength is the maximum length of a link. D-HELPER refuses a longer link.
	maxLinkLength = 4096
)

// linkError is a refused link. Its text is the short French reason that the user sees.
type linkError struct {
	reason string
}

func (e *linkError) Error() string {
	return e.reason
}

// refuse makes a linkError with a short French reason.
func refuse(reason string) error {
	return &linkError{reason: reason}
}

// errFolderNotFound tells that the folder of the link does not exist on this computer.
var errFolderNotFound = errors.New("folder not found")

// isLink tells if an argument is a d-helper link.
func isLink(arg string) bool {
	return strings.HasPrefix(strings.ToLower(arg), linkScheme+":")
}

// parseLink reads a link and gives the relative path of the folder.
// The relative path uses "/" as separator.
// The link must be: d-helper://open?path=<encoded relative path>.
func parseLink(raw string) (string, error) {
	if len(raw) > maxLinkLength {
		return "", refuse("lien trop long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", refuse("lien illisible")
	}
	// url.Parse sets the scheme in lower case.
	if u.Scheme != linkScheme {
		return "", refuse("schéma inconnu")
	}
	if u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return "", refuse("lien illisible")
	}
	// Some browsers add a "/" after the host. Accept it.
	if u.Host != linkActionOpen || (u.Path != "" && u.Path != "/") {
		return "", refuse("action inconnue")
	}

	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", refuse("paramètres illisibles")
	}
	for name := range values {
		if name != linkParamPath {
			return "", refuse("paramètre inconnu")
		}
	}
	paths := values[linkParamPath]
	if len(paths) == 0 {
		return "", refuse("chemin absent")
	}
	if len(paths) > 1 {
		return "", refuse("paramètre path répété")
	}

	rel := paths[0]
	if err := checkRelativePath(rel); err != nil {
		return "", err
	}
	return rel, nil
}

// checkRelativePath makes sure that a relative path is safe on Windows.
// It refuses absolute paths, "." and ".." segments, empty segments,
// forbidden characters and the names that Windows reserves.
func checkRelativePath(rel string) error {
	if rel == "" {
		return refuse("chemin vide")
	}
	if !utf8.ValidString(rel) {
		return refuse("caractère interdit")
	}
	if strings.HasPrefix(rel, "/") {
		return refuse("chemin absolu")
	}
	for _, r := range rel {
		switch {
		case unicode.IsControl(r):
			// This includes the NUL character.
			return refuse("caractère de contrôle")
		case r == '\\':
			return refuse("caractère \\ interdit")
		case r == ':':
			return refuse("caractère : interdit")
		case strings.ContainsRune(`<>"|?*`, r):
			return refuse("caractère interdit")
		}
	}
	for _, segment := range strings.Split(rel, "/") {
		switch {
		case segment == "":
			return refuse("segment vide")
		case segment == "." || segment == "..":
			return refuse("segment . ou .. interdit")
		case strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " "):
			// Windows removes a final dot or space. Thus the name can change.
			return refuse("nom terminé par un point ou un espace")
		case isReservedName(segment):
			return refuse("nom réservé par Windows")
		}
	}
	return nil
}

// isReservedName tells if a name is a device name that Windows reserves.
// Examples: CON, NUL, COM1, LPT1, con.txt.
func isReservedName(segment string) bool {
	name := strings.ToUpper(segment)
	if i := strings.IndexByte(name, '.'); i >= 0 {
		name = name[:i]
	}
	name = strings.TrimRight(name, " ")
	switch name {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if strings.HasPrefix(name, "COM") || strings.HasPrefix(name, "LPT") {
		number := name[3:]
		return utf8.RuneCountInString(number) == 1 && strings.Contains("0123456789¹²³", number)
	}
	return false
}

// resolveFolder joins the relative path to the root and gives the real path of the folder.
// It resolves the symbolic links and the junctions.
// It makes sure that the result is a folder in the root.
func resolveFolder(root, rel string) (string, error) {
	realRoot, err := realPath(root)
	if err != nil {
		return "", errRootNotFound
	}
	info, err := os.Stat(realRoot)
	if err != nil || !info.IsDir() {
		return "", errRootNotFound
	}

	target := filepath.Join(realRoot, filepath.FromSlash(rel))
	if !isInside(realRoot, target) {
		return "", refuse("le chemin sort du dossier Dahotecc")
	}

	realTarget, err := realPath(target)
	if err != nil {
		return "", errFolderNotFound
	}
	// Do the check again: a link or a junction can go out of the root.
	if !isInside(realRoot, realTarget) {
		return "", refuse("le chemin sort du dossier Dahotecc")
	}

	info, err = os.Stat(realTarget)
	if err != nil {
		return "", errFolderNotFound
	}
	if !info.IsDir() {
		return "", refuse("la cible n'est pas un dossier")
	}
	return realTarget, nil
}

// isInside tells if path is root or is in root.
func isInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
