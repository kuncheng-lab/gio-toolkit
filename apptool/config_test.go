package apptool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigPathPrefersWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("test: true"), 0o644); err != nil {
		t.Fatal(err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if got := DefaultConfigPath("demo"); got != "config.yaml" {
		t.Fatalf("DefaultConfigPath = %q, want %q", got, "config.yaml")
	}
}

func TestDefaultConfigPathFallsBackToAppDirectory(t *testing.T) {
	dir := t.TempDir()

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(original)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	want := filepath.Join("apps", "demo", "config.yaml")
	if got := DefaultConfigPath("demo"); got != want {
		t.Fatalf("DefaultConfigPath = %q, want %q", got, want)
	}
}
