package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeInfoFile writes a Dropbox info.json file with the given content.
// The extra fields are the same as in a real file. D-HELPER ignores them.
func writeInfoFile(t *testing.T, file string, accounts map[string]string) {
	t.Helper()
	content := map[string]any{}
	for name, path := range accounts {
		content[name] = map[string]any{
			"path":              path,
			"host":              123456789,
			"is_team":           name == "business",
			"subscription_type": "Business",
		}
	}
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	mustMkdir(t, filepath.Dir(file))
	if err := os.WriteFile(file, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRootFromInfoFile(t *testing.T) {
	business := filepath.Join(t.TempDir(), "Dropbox (Dahotecc)")
	personal := filepath.Join(t.TempDir(), "Dropbox")

	tests := []struct {
		name     string
		accounts map[string]string
		want     string
	}{
		{"business has priority", map[string]string{"business": business, "personal": personal}, filepath.Join(business, "Dahotecc")},
		{"personal only", map[string]string{"personal": personal}, filepath.Join(personal, "Dahotecc")},
		{"empty business path", map[string]string{"business": "", "personal": personal}, filepath.Join(personal, "Dahotecc")},
	}
	for _, tt := range tests {
		file := filepath.Join(t.TempDir(), "info.json")
		writeInfoFile(t, file, tt.accounts)
		got, err := rootFromInfoFile(file)
		if err != nil {
			t.Errorf("%s: unexpected error %v", tt.name, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRootFromInfoFileNotFound(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		file := filepath.Join(dir, name)
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return file
	}
	files := map[string]string{
		"no account":    write("vide.json", `{}`),
		"bad JSON":      write("casse.json", `{"business":`),
		"relative path": write("relatif.json", `{"personal":{"path":"Dropbox"}}`),
		"missing file":  filepath.Join(dir, "absent.json"),
	}
	for name, file := range files {
		if _, err := rootFromInfoFile(file); !errors.Is(err, errRootNotFound) {
			t.Errorf("%s: got %v, want errRootNotFound", name, err)
		}
	}
}

func TestFindRootReadsInfoFile(t *testing.T) {
	localAppData := t.TempDir()
	business := filepath.Join(t.TempDir(), "Dropbox (Dahotecc)")
	writeInfoFile(t, filepath.Join(localAppData, "Dropbox", "info.json"), map[string]string{"business": business})
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv(envRoot, "")

	got, err := findRoot()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if want := filepath.Join(business, "Dahotecc"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindRootUsesVariableFirst(t *testing.T) {
	localAppData := t.TempDir()
	writeInfoFile(t, filepath.Join(localAppData, "Dropbox", "info.json"), map[string]string{"personal": t.TempDir()})
	root := filepath.Join(t.TempDir(), "Dahotecc")
	t.Setenv("LOCALAPPDATA", localAppData)
	t.Setenv(envRoot, root+string(filepath.Separator))

	got, err := findRoot()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if got != root {
		t.Errorf("got %q, want %q", got, root)
	}
}

func TestFindRootNotFound(t *testing.T) {
	t.Setenv(envRoot, "relatif/Dahotecc")
	if _, err := findRoot(); !errors.Is(err, errRootNotFound) {
		t.Errorf("relative variable: got %v, want errRootNotFound", err)
	}

	t.Setenv(envRoot, "")
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if _, err := findRoot(); !errors.Is(err, errRootNotFound) {
		t.Errorf("no info.json: got %v, want errRootNotFound", err)
	}

	t.Setenv("LOCALAPPDATA", "")
	if _, err := findRoot(); !errors.Is(err, errRootNotFound) {
		t.Errorf("no LOCALAPPDATA: got %v, want errRootNotFound", err)
	}
}
