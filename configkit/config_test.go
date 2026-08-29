package configkit

import (
	"os"
	"path/filepath"
	"testing"
)

type testConfig struct {
	Addr string `yaml:"addr"`
	Size int    `yaml:"size"`
}

func defaultTestConfig() testConfig {
	return testConfig{Addr: "127.0.0.1:2333", Size: 100}
}

func TestLoadCreatesFileWithDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")
	cfg := defaultTestConfig()

	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if cfg.Addr != "127.0.0.1:2333" || cfg.Size != 100 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadOverridesOnlyPresentFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("size: 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultTestConfig()
	if err := Load(path, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Size != 42 {
		t.Fatalf("size = %d, want 42", cfg.Size)
	}
	if cfg.Addr != "127.0.0.1:2333" {
		t.Fatalf("addr = %q, want default", cfg.Addr)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	want := testConfig{Addr: "10.0.0.1:8080", Size: 7}

	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	var got testConfig
	if err := Load(path, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got = %+v, want %+v", got, want)
	}
}

func TestLoadInvalidYAMLReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("size: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := defaultTestConfig()
	if err := Load(path, &cfg); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
