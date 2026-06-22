package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
)

// settings is the fully-resolved send configuration after merging, in order of
// precedence: command-line flags > environment variables > config-file profile.
type settings struct {
	baseURL     string // "" → client uses DefaultBaseURL
	hookID      string
	defaultType string // event type used when --type is not given
}

// resolveSettings merges flag values, environment variables and the selected
// profile into the effective settings.
//
// Per-field precedence:
//   - base URL:   --url > HOOKTAP_BASE_URL > host of a full webhook URL
//   - webhook id: --hook > HOOKTAP_HOOK_ID > HOOKTAP_WEBHOOK_URL > profile.HookID > profile.URL
//   - type:       --type > profile.Type > client.DefaultType
func resolveSettings(flagURL, flagHook, flagType string, prof config.Profile) (settings, error) {
	var s settings
	s.baseURL = firstNonEmpty(flagURL, os.Getenv("HOOKTAP_BASE_URL"))

	source := firstNonEmpty(
		flagHook,
		os.Getenv("HOOKTAP_HOOK_ID"),
		os.Getenv("HOOKTAP_WEBHOOK_URL"),
		prof.HookID,
		prof.URL,
	)
	if source == "" {
		return settings{}, fmt.Errorf("%w: no webhook configured — pass --hook, set HOOKTAP_HOOK_ID, or run 'hooktap config set hook_id <id>'", errUsage)
	}

	base, id := splitWebhook(source)
	if s.baseURL == "" {
		s.baseURL = base
	}
	s.hookID = id
	if s.hookID == "" {
		return settings{}, fmt.Errorf("%w: could not determine webhook id from %q", errUsage, source)
	}

	s.defaultType = firstNonEmpty(flagType, prof.Type, client.DefaultType)
	return s, nil
}

// splitWebhook accepts either a bare webhook id or a full ".../webhook/{id}"
// URL and returns the base URL (empty for a bare id) and the id.
func splitWebhook(s string) (base, id string) {
	if i := strings.Index(s, "/webhook/"); i != -1 {
		return s[:i], strings.Trim(s[i+len("/webhook/"):], "/")
	}
	return "", s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
