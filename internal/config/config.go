// Package config loads and persists the user's HookTap CLI configuration:
// a TOML file holding named profiles, each pointing at a webhook. It is
// deliberately free of any env/flag logic — merging precedence lives in cmd —
// so it stays a pure file<->struct mapping that is easy to test.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
)

// DefaultProfileName is used when the file declares no `default` and the user
// passes no --profile.
const DefaultProfileName = "default"

// Config is the on-disk shape of config.toml.
type Config struct {
	// Default is the profile used when --profile is omitted.
	Default string `toml:"default,omitempty"`
	// Profiles maps a profile name to its settings.
	Profiles map[string]Profile `toml:"profiles,omitempty"`
}

// Profile is one named webhook target.
type Profile struct {
	HookID string `toml:"hook_id,omitempty"`
	URL    string `toml:"url,omitempty"`
	Type   string `toml:"type,omitempty"`
}

// Path returns the config file location, honouring XDG_CONFIG_HOME and falling
// back to ~/.config/hooktap/config.toml.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "hooktap", "config.toml"), nil
}

// Load reads the config file. A missing file is not an error: it returns an
// empty, ready-to-use Config so first-run commands work without setup.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	cfg := &Config{Profiles: map[string]Profile{}}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

// Save writes cfg to disk, creating the directory if needed. The file is
// written 0600 because it can contain a webhook id, which acts as a bearer
// token in the HookTap API.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

// ResolveName returns the effective profile name: the explicit request, else
// the file default, else DefaultProfileName.
func (c *Config) ResolveName(requested string) string {
	if requested != "" {
		return requested
	}
	if c.Default != "" {
		return c.Default
	}
	return DefaultProfileName
}

// Profile returns the named profile (zero value if absent — callers treat an
// empty profile as "nothing configured").
func (c *Config) Profile(name string) Profile {
	return c.Profiles[name]
}

// ProfileNames returns the configured profile names, sorted.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
