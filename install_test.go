package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPath(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	got, err := installPath()
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if want := filepath.Join(localAppData, "Programs", "d-helper", "d-helper.exe"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	t.Setenv("LOCALAPPDATA", "")
	if _, err := installPath(); err == nil {
		t.Error("empty LOCALAPPDATA: want an error")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.exe")
	dest := filepath.Join(dir, "dest.exe")
	if err := os.WriteFile(source, []byte("nouveau"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("ancien"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(source, dest); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "nouveau" {
		t.Errorf("dest = %q, %v: want the new content", data, err)
	}
	if _, err := os.Stat(dest + ".new"); !os.IsNotExist(err) {
		t.Error("the temporary file must not stay")
	}
	if !isSameFile(dest, dest) || isSameFile(source, dest) {
		t.Error("isSameFile: bad result")
	}
}
