package cmd

import (
	"testing"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
)

func TestResolveContent(t *testing.T) {
	const def = "Notification from host"
	tests := []struct {
		name      string
		titleArg  string
		titleFlag string
		bodyFlag  string
		stdin     string
		hasStdin  bool
		wantTitle string
		wantBody  string
		wantErr   bool
	}{
		{
			name:      "title arg only",
			titleArg:  "Build done",
			wantTitle: "Build done",
			wantBody:  "",
		},
		{
			name:      "stdin becomes body, default title",
			stdin:     "Staging is live\n",
			hasStdin:  true,
			wantTitle: def,
			wantBody:  "Staging is live",
		},
		{
			name:      "title arg plus stdin body",
			titleArg:  "Deploy",
			stdin:     "  Prod is up  ",
			hasStdin:  true,
			wantTitle: "Deploy",
			wantBody:  "Prod is up",
		},
		{
			name:      "body flag overrides stdin",
			titleArg:  "T",
			bodyFlag:  "explicit body",
			stdin:     "ignored",
			hasStdin:  true,
			wantTitle: "T",
			wantBody:  "explicit body",
		},
		{
			name:      "title flag when no arg",
			titleFlag: "From flag",
			wantTitle: "From flag",
		},
		{
			name:     "nothing provided errors",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body, err := resolveContent(tt.titleArg, tt.titleFlag, tt.bodyFlag, def, tt.stdin, tt.hasStdin)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tt.wantTitle {
				t.Errorf("title = %q, want %q", title, tt.wantTitle)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestResolveSettings(t *testing.T) {
	tests := []struct {
		name      string
		flagURL   string
		flagHook  string
		flagType  string
		env       map[string]string
		prof      config.Profile
		wantBase  string
		wantID    string
		wantType  string
		wantErr   bool
	}{
		{
			name:     "plain id via flag",
			flagHook: "abc12345",
			wantID:   "abc12345",
			wantType: client.DefaultType,
		},
		{
			name:     "full url via flag splits base and id",
			flagHook: "https://hooks.hooktap.me/webhook/abc12345",
			wantBase: "https://hooks.hooktap.me",
			wantID:   "abc12345",
			wantType: client.DefaultType,
		},
		{
			name:     "url flag overrides base from full hook url",
			flagURL:  "https://staging.example.com",
			flagHook: "https://hooks.hooktap.me/webhook/abc12345",
			wantBase: "https://staging.example.com",
			wantID:   "abc12345",
			wantType: client.DefaultType,
		},
		{
			name:     "HOOKTAP_HOOK_ID env",
			env:      map[string]string{"HOOKTAP_HOOK_ID": "envid123"},
			wantID:   "envid123",
			wantType: client.DefaultType,
		},
		{
			name:     "HOOKTAP_WEBHOOK_URL env splits",
			env:      map[string]string{"HOOKTAP_WEBHOOK_URL": "https://hooks.hooktap.me/webhook/envurl9"},
			wantBase: "https://hooks.hooktap.me",
			wantID:   "envurl9",
			wantType: client.DefaultType,
		},
		{
			name:     "profile provides id and type",
			prof:     config.Profile{HookID: "profid", Type: client.TypeFeed},
			wantID:   "profid",
			wantType: client.TypeFeed,
		},
		{
			name:     "flag beats env beats profile",
			flagHook: "flagid",
			env:      map[string]string{"HOOKTAP_HOOK_ID": "envid"},
			prof:     config.Profile{HookID: "profid"},
			wantID:   "flagid",
			wantType: client.DefaultType,
		},
		{
			name:     "flag type overrides profile type",
			flagHook: "id",
			flagType: client.TypeWidget,
			prof:     config.Profile{Type: client.TypeFeed},
			wantID:   "id",
			wantType: client.TypeWidget,
		},
		{
			name:    "nothing configured errors",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range []string{"HOOKTAP_HOOK_ID", "HOOKTAP_WEBHOOK_URL", "HOOKTAP_BASE_URL"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			s, err := resolveSettings(tt.flagURL, tt.flagHook, tt.flagType, tt.prof)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.baseURL != tt.wantBase {
				t.Errorf("baseURL = %q, want %q", s.baseURL, tt.wantBase)
			}
			if s.hookID != tt.wantID {
				t.Errorf("hookID = %q, want %q", s.hookID, tt.wantID)
			}
			if s.defaultType != tt.wantType {
				t.Errorf("defaultType = %q, want %q", s.defaultType, tt.wantType)
			}
		})
	}
}
