package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{refuse("chemin absolu"), "Lien D-HELPER refusé : chemin absolu."},
		{errRootNotFound, "D-HELPER ne trouve pas le dossier Dropbox Dahotecc sur ce poste."},
		{errFolderNotFound, "Dossier introuvable : Devis 2026/A. Il a peut-être été déplacé ; relisez le Dropbox dans D-HUB."},
		{errors.New("x"), "Erreur D-HELPER : x"},
	}
	for _, tt := range tests {
		if got := errorMessage(tt.err, "Devis 2026/A"); got != tt.want {
			t.Errorf("errorMessage(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

func TestRunCheck(t *testing.T) {
	root, _ := makeTestRoot(t)
	t.Setenv(envRoot, root)

	tests := []struct {
		link string
		want int
	}{
		{"d-helper://open?path=Devis%202026%2FClient", 0},
		{"d-helper://open?path=..%2Foutside", 1},
		{"d-helper://open?path=sortie", 1},
		{"d-helper://open?path=notes.txt", 1},
		{"d-helper://open?path=absent", 1},
	}
	for _, tt := range tests {
		if got := run([]string{"--check", tt.link}); got != tt.want {
			t.Errorf("run --check %q = %d, want %d", tt.link, got, tt.want)
		}
	}

	t.Setenv(envRoot, filepath.Join(root, "absent"))
	if got := run([]string{"--check", "d-helper://open?path=Devis%202026"}); got != 1 {
		t.Errorf("missing root: got %d, want 1", got)
	}
}

func TestRunUnknownArguments(t *testing.T) {
	for _, args := range [][]string{{"--inconnu"}, {"--install", "x"}, {"d-helper://open?path=a", "b"}} {
		if got := run(args); got != 2 {
			t.Errorf("run(%q) = %d, want 2", args, got)
		}
	}
}
