package config

import (
	"path/filepath"
	"testing"
)

// withTempConfig points Path() at a temp dir via XDG_CONFIG_HOME.
func withTempConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "hooktap", "config.toml")
}

func TestLoadMissingReturnsEmpty(t *testing.T) {
	withTempConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if cfg.Profiles == nil {
		t.Error("Profiles map should be initialised, not nil")
	}
	if len(cfg.ProfileNames()) != 0 {
		t.Errorf("expected no profiles, got %v", cfg.ProfileNames())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withTempConfig(t)

	cfg := &Config{
		Default: "ci",
		Profiles: map[string]Profile{
			"personal": {HookID: "abc123"},
			"ci":       {URL: "https://hooks.hooktap.me/webhook/xyz", Type: "push"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Default != "ci" {
		t.Errorf("Default = %q, want ci", got.Default)
	}
	if got.Profile("personal").HookID != "abc123" {
		t.Errorf("personal.HookID = %q, want abc123", got.Profile("personal").HookID)
	}
	if got.Profile("ci").Type != "push" {
		t.Errorf("ci.Type = %q, want push", got.Profile("ci").Type)
	}
}

func TestResolveName(t *testing.T) {
	cfg := &Config{Default: "ci", Profiles: map[string]Profile{}}
	if got := cfg.ResolveName("explicit"); got != "explicit" {
		t.Errorf("explicit request: got %q", got)
	}
	if got := cfg.ResolveName(""); got != "ci" {
		t.Errorf("default request: got %q, want ci", got)
	}
	empty := &Config{}
	if got := empty.ResolveName(""); got != DefaultProfileName {
		t.Errorf("no default: got %q, want %q", got, DefaultProfileName)
	}
}
