package client

// Event types accepted by the HookTap webhook endpoint.
// The server silently falls back to "feed" for any unknown value, so the CLI
// validates client-side to give users an immediate, clear error instead.
const (
	TypePush   = "push"
	TypeFeed   = "feed"
	TypeWidget = "widget"
)

// DefaultType is what the CLI sends when the user does not specify --type.
// We default to "push" to match the existing hooktap-integrations helpers and
// the most intuitive CLI expectation (running a command should notify you).
const DefaultType = TypePush

// validTypes is the set of types the webhook endpoint understands.
var validTypes = map[string]struct{}{
	TypePush:   {},
	TypeFeed:   {},
	TypeWidget: {},
}

// ValidType reports whether t is a type the HookTap endpoint accepts.
func ValidType(t string) bool {
	_, ok := validTypes[t]
	return ok
}

// Payload is the JSON body sent to POST /webhook/{webhookId}.
//
// It mirrors the contract in HookTap/functions/index.js:
//   - Type is one of push|feed|widget.
//   - Title is required (the server rejects an empty title with 400).
//   - Body, Extra ("payload") and DeepLink are optional.
type Payload struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Body     string         `json:"body,omitempty"`
	Extra    map[string]any `json:"payload,omitempty"`
	DeepLink string         `json:"deepLink,omitempty"`
}
