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

func TestChooseAction(t *testing.T) {
	installDir := filepath.Join("home", "Programs", "d-helper")
	installed := filepath.Join(installDir, "d-helper.exe")
	downloaded := filepath.Join("home", "Downloads", "d-helper.exe")
	tests := []struct {
		name      string
		running   string
		dir       string
		installed bool
		want      startAction
	}{
		{"first install", downloaded, installDir, false, actionInstall},
		{"new version downloaded", downloaded, installDir, true, actionUpdate},
		{"installed program", installed, installDir, true, actionRepair},
		{"other case in the path", filepath.Join("HOME", "programs", "D-HELPER", "d-helper.exe"), installDir, true, actionRepair},
		{"install folder with a final separator", installed, installDir + string(filepath.Separator), true, actionRepair},
		{"install folder without the program", installed, installDir, false, actionInstall},
		{"sub-folder of the install folder", filepath.Join(installDir, "x", "d-helper.exe"), installDir, true, actionUpdate},
	}
	for _, tt := range tests {
		if got := chooseAction(tt.running, tt.dir, tt.installed); got != tt.want {
			t.Errorf("%s: chooseAction = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestIsInstalled(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "d-helper.exe")
	if isInstalled(exe) {
		t.Error("missing file: want false")
	}
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !isInstalled(exe) {
		t.Error("file: want true")
	}
	if isInstalled(dir) {
		t.Error("folder: want false")
	}
}

// A destination that the first rename cannot replace (here a folder) is moved to dest.old.
// On Windows, this is the case of a running program.
func TestReplaceFileMovesOldCopy(t *testing.T) {
	dir := t.TempDir()
	temp := filepath.Join(dir, "d-helper.exe.new")
	dest := filepath.Join(dir, "d-helper.exe")
	if err := os.WriteFile(temp, []byte("nouveau"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceFile(temp, dest); err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "nouveau" {
		t.Errorf("dest = %q, %v: want the new content", data, err)
	}
	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Error("the old copy must not stay")
	}
}

// If D-HELPER cannot rename the files, it gives errInstalledBusy and keeps the installed program.
func TestReplaceFileBusy(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores the permissions of the folder")
	}
	dir := t.TempDir()
	temp := filepath.Join(dir, "d-helper.exe.new")
	dest := filepath.Join(dir, "d-helper.exe")
	if err := os.WriteFile(temp, []byte("nouveau"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("ancien"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A read-only folder does not let D-HELPER rename the files.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := replaceFile(temp, dest); err != errInstalledBusy {
		t.Fatalf("got %v, want errInstalledBusy", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "ancien" {
		t.Errorf("dest = %q, %v: want the installed program", data, err)
	}
}
