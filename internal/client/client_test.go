package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidType(t *testing.T) {
	for _, ok := range []string{TypePush, TypeFeed, TypeWidget} {
		if !ValidType(ok) {
			t.Errorf("ValidType(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"build", "", "PUSH", "notify"} {
		if ValidType(bad) {
			t.Errorf("ValidType(%q) = true, want false", bad)
		}
	}
}

func TestSend_Success_BuildsContract(t *testing.T) {
	var gotPath, gotCT string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"type":"push","userId":"u1","eventId":"e1","notificationSent":true,"notificationsSent":2,"notificationsTotal":2}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.Send(context.Background(), "abc12345", Payload{
		Type:  TypePush,
		Title: "CI succeeded",
		Body:  "Staging deploy is live",
		Extra: map[string]any{"branch": "main"},
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if gotPath != "/webhook/abc12345" {
		t.Errorf("path = %q, want /webhook/abc12345", gotPath)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}
	// The JSON key for Extra must be "payload" to match the server contract.
	if _, ok := gotBody["payload"]; !ok {
		t.Errorf("request body missing %q key: %v", "payload", gotBody)
	}
	if gotBody["title"] != "CI succeeded" {
		t.Errorf("title = %v, want CI succeeded", gotBody["title"])
	}
	if !resp.Success || resp.EventID != "e1" || resp.NotificationsSent != 2 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestSend_OmitsEmptyOptionalFields(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Send(context.Background(), "id", Payload{Type: TypeFeed, Title: "x"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"body", "payload", "deepLink"} {
		if _, present := gotBody[k]; present {
			t.Errorf("empty optional field %q should be omitted, body=%v", k, gotBody)
		}
	}
}

func TestSend_StatusErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		{"rate limited", http.StatusTooManyRequests, `{"success":false,"error":"Rate limit exceeded. Max 1 request per second."}`, ErrRateLimited},
		{"not found", http.StatusNotFound, `{"success":false,"error":"No webhook found"}`, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			_, err := New(srv.URL).Send(context.Background(), "id", Payload{Type: TypePush, Title: "x"})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("err = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
		})
	}
}

func TestSend_BadRequestSurfacesServerMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"success":false,"error":"\"title\" must be a non-empty string"}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Send(context.Background(), "id", Payload{Type: TypePush})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if got := err.Error(); got == "" || !contains(got, "title") || !contains(got, "400") {
		t.Errorf("error %q should mention the server message and status", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
