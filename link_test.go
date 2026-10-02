package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLinkAcceptsValidLinks(t *testing.T) {
	tests := []struct {
		link string
		want string
	}{
		// Encoded as the browser does with encodeURIComponent.
		{"d-helper://open?path=Devis%202026%2F260915-Client-Nom%2FFAB", "Devis 2026/260915-Client-Nom/FAB"},
		{"d-helper://open?path=Projets%2FCaf%C3%A9%20%C3%A9t%C3%A9", "Projets/Café été"},
		{"d-helper://open?path=A%20%26%20B%2FN%C2%B0%20%231", "A & B/N° #1"},
		{"d-helper://open?path=Plan%20(v2)", "Plan (v2)"},
		{"d-helper://open?path=Plan%20%28v2%29", "Plan (v2)"},
		{"d-helper://open?path=a%2Bb", "a+b"},
		{"d-helper://open?path=v1.2", "v1.2"},
		{"d-helper://open?path=CONSOLE", "CONSOLE"},
		{"d-helper://open?path=COM10", "COM10"},
		// Some browsers add a "/" after the host.
		{"d-helper://open/?path=Devis%202026", "Devis 2026"},
		// url.Parse sets the scheme in lower case.
		{"D-HELPER://open?path=Devis", "Devis"},
	}
	for _, tt := range tests {
		got, err := parseLink(tt.link)
		if err != nil {
			t.Errorf("parseLink(%q): unexpected error %v", tt.link, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseLink(%q) = %q, want %q", tt.link, got, tt.want)
		}
	}
}

func TestParseLinkRefusesBadLinks(t *testing.T) {
	links := []string{
		// Segments "." and "..".
		"d-helper://open?path=..",
		"d-helper://open?path=%2E%2E%2Fsecret",
		"d-helper://open?path=a%2F..%2Fb",
		"d-helper://open?path=.",
		"d-helper://open?path=a%2F.%2Fb",
		// Absolute paths.
		"d-helper://open?path=%2Fetc",
		"d-helper://open?path=%5CWindows",
		"d-helper://open?path=C%3A",
		"d-helper://open?path=C%3A%2FWindows",
		"d-helper://open?path=C%3A%5CWindows",
		// UNC paths.
		"d-helper://open?path=%5C%5Cserveur%5Cpartage",
		"d-helper://open?path=%2F%2Fserveur%2Fpartage",
		// Forbidden characters.
		"d-helper://open?path=a%5Cb",
		"d-helper://open?path=a%00b",
		"d-helper://open?path=a%0Ab",
		"d-helper://open?path=a%3Fb",
		"d-helper://open?path=a%2A",
		"d-helper://open?path=a%22b",
		"d-helper://open?path=a%3Cb",
		"d-helper://open?path=%FF",
		// Empty path or empty segment.
		"d-helper://open?path=",
		"d-helper://open?path=a%2F%2Fb",
		"d-helper://open?path=a%2F",
		// Final dot or space.
		"d-helper://open?path=dossier.",
		"d-helper://open?path=dossier%20",
		// Names that Windows reserves.
		"d-helper://open?path=CON",
		"d-helper://open?path=nul",
		"d-helper://open?path=COM1",
		"d-helper://open?path=a%2FLPT9%2Fb",
		"d-helper://open?path=con.txt",
		"d-helper://open?path=AUX",
		"d-helper://open?path=PRN",
		"d-helper://open?path=COM%C2%B9",
		// Unknown action or bad host.
		"d-helper://close?path=a",
		"d-helper://OPEN?path=a",
		"d-helper://open/x?path=a",
		"d-helper://open:80?path=a",
		"d-helper://user@open?path=a",
		"d-helper:open?path=a",
		// Missing, unknown or repeated parameter.
		"d-helper://open",
		"d-helper://open?chemin=a",
		"d-helper://open?path=a&x=1",
		"d-helper://open?path=a&=1",
		"d-helper://open?path=a&path=b",
		"d-helper://open?path=a;b",
		"d-helper://open?path=%ZZ",
		// Not encoded "#": the end of the path becomes a fragment.
		"d-helper://open?path=a#b",
		// Bad scheme.
		"https://open?path=a",
		"file://open?path=a",
		"d-helperx://open?path=a",
		"open?path=a",
		// Too long.
		"d-helper://open?path=" + strings.Repeat("a", maxLinkLength),
	}
	for _, link := range links {
		rel, err := parseLink(link)
		var refused *linkError
		if !errors.As(err, &refused) {
			t.Errorf("parseLink(%q) = %q, %v: want a refused link", link, rel, err)
		}
	}
}

func TestIsLink(t *testing.T) {
	if !isLink("d-helper://open?path=a") || !isLink("D-Helper:x") {
		t.Error("isLink: want true for a d-helper link")
	}
	if isLink("--install") || isLink("https://example.com") {
		t.Error("isLink: want false for other arguments")
	}
}

// makeTestRoot makes a root folder with this content:
//
//	Devis 2026/Client/      folder
//	notes.txt               file
//	raccourci -> Devis 2026 symbolic link in the root
//	sortie -> <outside>     symbolic link out of the root
//	casse -> <absent>       symbolic link to a missing folder
//
// It gives the root and the outside folder.
func makeTestRoot(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "Dahotecc")
	outside = filepath.Join(base, "outside")
	mustMkdir(t, filepath.Join(root, "Devis 2026", "Client"))
	mustMkdir(t, filepath.Join(outside, "secret"))
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustSymlink(t, filepath.Join(root, "Devis 2026"), filepath.Join(root, "raccourci"))
	mustSymlink(t, outside, filepath.Join(root, "sortie"))
	mustSymlink(t, filepath.Join(base, "absent"), filepath.Join(root, "casse"))
	return root, outside
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func TestResolveFolderAcceptsFoldersInRoot(t *testing.T) {
	root, _ := makeTestRoot(t)
	want := filepath.Join(root, "Devis 2026", "Client")

	for _, rel := range []string{"Devis 2026/Client", "raccourci/Client"} {
		got, err := resolveFolder(root, rel)
		if err != nil {
			t.Errorf("resolveFolder(%q): unexpected error %v", rel, err)
			continue
		}
		if got != want {
			t.Errorf("resolveFolder(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestResolveFolderAcceptsRootThroughLink(t *testing.T) {
	root, _ := makeTestRoot(t)
	rootLink := filepath.Join(t.TempDir(), "lien-racine")
	mustSymlink(t, root, rootLink)

	got, err := resolveFolder(rootLink, "Devis 2026")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if want := filepath.Join(root, "Devis 2026"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveFolderRefusesLinkOutOfRoot(t *testing.T) {
	root, _ := makeTestRoot(t)
	for _, rel := range []string{"sortie", "sortie/secret"} {
		_, err := resolveFolder(root, rel)
		var refused *linkError
		if !errors.As(err, &refused) {
			t.Errorf("resolveFolder(%q) = %v: want a refused link", rel, err)
		}
	}
}

func TestResolveFolderRefusesFile(t *testing.T) {
	root, _ := makeTestRoot(t)
	_, err := resolveFolder(root, "notes.txt")
	var refused *linkError
	if !errors.As(err, &refused) {
		t.Errorf("got %v: want a refused link", err)
	}
}

func TestResolveFolderMissingFolder(t *testing.T) {
	root, _ := makeTestRoot(t)
	for _, rel := range []string{"absent", "Devis 2026/absent", "casse", "notes.txt/x"} {
		_, err := resolveFolder(root, rel)
		if !errors.Is(err, errFolderNotFound) {
			t.Errorf("resolveFolder(%q) = %v: want errFolderNotFound", rel, err)
		}
	}
}

func TestResolveFolderMissingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "absent")
	_, err := resolveFolder(root, "Devis 2026")
	if !errors.Is(err, errRootNotFound) {
		t.Errorf("got %v: want errRootNotFound", err)
	}
}

func TestIsInside(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "racine")
	tests := []struct {
		path string
		want bool
	}{
		{root, true},
		{filepath.Join(root, "a", "b"), true},
		{filepath.Join(root, "..a"), true},
		{filepath.Dir(root), false},
		{filepath.Join(root, "..", "autre"), false},
		{root + "2", false},
	}
	for _, tt := range tests {
		if got := isInside(root, tt.path); got != tt.want {
			t.Errorf("isInside(%q, %q) = %v, want %v", root, tt.path, got, tt.want)
		}
	}
}
