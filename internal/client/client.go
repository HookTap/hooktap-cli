// Package client is the reusable core for talking to the HookTap webhook
// endpoint. Every CLI command builds on Send; it has no dependency on cobra or
// any UI concern so it can also be imported as a library.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the production HookTap ingest host. Overridable for
// staging/self-hosting via the --url flag or HOOKTAP_BASE_URL.
const DefaultBaseURL = "https://hooks.hooktap.me"

const defaultTimeout = 10 * time.Second

// Sentinel errors let the cmd layer map failures to distinct exit codes
// without string matching on the response body.
var (
	// ErrRateLimited is returned on HTTP 429. The endpoint allows max 1
	// request per second per webhook; callers may retry after a short wait.
	ErrRateLimited = errors.New("rate limit exceeded (max 1 request/second)")
	// ErrNotFound is returned on HTTP 404 — the webhookId is unknown.
	ErrNotFound = errors.New("unknown webhookId")
)

// Client sends events to a HookTap webhook.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a Client with sane defaults. baseURL may be empty to use
// DefaultBaseURL.
func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: defaultTimeout},
	}
}

// Response is the decoded JSON returned by the webhook endpoint.
type Response struct {
	Success            bool   `json:"success"`
	Type               string `json:"type"`
	UserID             string `json:"userId"`
	EventID            string `json:"eventId"`
	Error              string `json:"error"`
	NotificationSent   bool   `json:"notificationSent"`
	NotificationsSent  int    `json:"notificationsSent"`
	NotificationsTotal int    `json:"notificationsTotal"`
}

// WebhookURL builds the full POST URL for a given webhook id.
func (c *Client) WebhookURL(hookID string) string {
	return c.BaseURL + "/webhook/" + hookID
}

// Send delivers p to the webhook identified by hookID.
//
// On success it returns the decoded Response. On a recognised HTTP error it
// returns one of the sentinel errors (wrapped) so the caller can branch on
// errors.Is. Validation of p.Title/p.Type is the caller's responsibility — see
// Payload.
func (c *Client) Send(ctx context.Context, hookID string, p Payload) (*Response, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return c.post(ctx, hookID, body)
}

// SendRaw posts an already-encoded JSON body to the webhook verbatim. It is
// used for --raw: the server applies its own field mapping (fieldMapping paths
// in HookTap/functions/index.js), so the CLI must not reshape the payload.
func (c *Client) SendRaw(ctx context.Context, hookID string, raw []byte) (*Response, error) {
	return c.post(ctx, hookID, raw)
}

// post sends body as application/json and maps the HTTP status to a Response or
// sentinel error.
func (c *Client) post(ctx context.Context, hookID string, body []byte) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL(hookID), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	// Decode best-effort; the body may be empty or non-JSON on edge cases.
	var out Response
	_ = json.Unmarshal(raw, &out)

	switch resp.StatusCode {
	case http.StatusOK:
		return &out, nil
	case http.StatusTooManyRequests:
		return &out, ErrRateLimited
	case http.StatusNotFound:
		return &out, fmt.Errorf("%w: %q", ErrNotFound, hookID)
	default:
		msg := out.Error
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = resp.Status
		}
		return &out, fmt.Errorf("hooktap: %s (HTTP %d)", msg, resp.StatusCode)
	}
}
