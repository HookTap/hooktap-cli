<div align="center">
  <img src="https://hooktap.me/hooktap-logo.png" alt="HookTap" height="40" />
  <h3>HookTap CLI</h3>
  <p>Send webhook events to the HookTap app straight from your terminal.</p>

  [![Homebrew](https://img.shields.io/badge/Homebrew-hooktap%2Ftap-orange?style=flat-square&logo=homebrew)](https://github.com/HookTap/homebrew-tap)
  [![Release](https://img.shields.io/github/v/release/HookTap/hooktap-cli?style=flat-square&logo=github)](https://github.com/HookTap/hooktap-cli/releases)
  [![App Store](https://img.shields.io/badge/App_Store-iOS-black?style=flat-square&logo=apple)](https://apps.apple.com/app/hooktap/id6670671021)
  [![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](./LICENSE)
</div>

---

HookTap turns HTTP POST requests into instant iPhone push notifications, lock screen widgets, and Live Activities. `hooktap` is the official command-line client: a single, dependency-free binary that follows Unix conventions — it reads from stdin, plays nicely with pipes, and exits with meaningful codes.

```bash
echo "Staging deploy is live" | hooktap send --title "Deploy"
```

## Install

**macOS / Linux** — Homebrew:

```bash
brew install HookTap/tap/hooktap
```

**Windows** — Scoop:

```powershell
scoop bucket add hooktap https://github.com/HookTap/scoop-bucket
scoop install hooktap
```

<details>
<summary>Other methods</summary>

```bash
# With Go
go install github.com/hooktap/hooktap-cli@latest

# Or download a prebuilt binary from the releases page:
# https://github.com/HookTap/hooktap-cli/releases
```
</details>

## Setup

Install [HookTap on the App Store](https://apps.apple.com/app/hooktap/id6670671021) and copy your webhook id (the `YOUR_HOOK_ID` part of `https://hooks.hooktap.me/webhook/YOUR_HOOK_ID`), then save it:

```bash
hooktap config set hook_id YOUR_HOOK_ID
hooktap ping        # ok — hooktap (2026-06-22T07:14:55.993Z)
```

## Usage

```bash
# Simplest form — the title is the argument
hooktap send "Build finished"

# Add a body and pick the event type
hooktap send "CI failed" --body "main branch" --type push

# Pipe text from stdin — it becomes the body
echo "Prod is up" | hooktap send --title "Deploy"
cat report.txt | hooktap send "Nightly report"

# Pipe a complete JSON body verbatim (for field-mapping webhooks)
generate-payload.sh | hooktap send --raw

# Machine-readable output for scripts
hooktap send "Order #123" --json | jq .eventId
```

### Event types

| Type | Behaviour |
|------|-----------|
| `push` *(default)* | Stores the event **and** sends an instant push notification |
| `feed` | Stores the event in the in-app feed only (no notification) |
| `widget` | Updates the lock screen / home screen widget |

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--title` | | Event title (alternative to the positional argument) |
| `--body` | `-b` | Event body text (overrides stdin) |
| `--type` | `-t` | `push` · `feed` · `widget` (default `push`) |
| `--raw` | | Send a complete JSON body read from stdin, verbatim |
| `--hook` | | Webhook id or full URL (overrides config/env) |
| `--profile` | `-p` | Config profile to use |
| `--url` | | Base URL override (staging / self-hosting) |
| `--json` | | Print the raw JSON response to stdout |

## Configuration

`hooktap` resolves each setting in this order: **command-line flag → environment variable → config file**.

### Config file & profiles

Stored at `~/.config/hooktap/config.toml` (honours `XDG_CONFIG_HOME`). Use named profiles to keep several webhooks side by side:

```bash
hooktap config set hook_id abc123                 # writes the default profile
hooktap config set url https://… --profile ci     # a named profile
hooktap config set type feed --profile ci
hooktap config use ci                             # change the default
hooktap config list                               # * marks the default
hooktap config path                               # print the file location
```

```toml
default = "ci"

[profiles.default]
hook_id = "abc123"

[profiles.ci]
url  = "https://hooks.hooktap.me/webhook/xyz"
type = "feed"
```

> The file is written `0600` — your webhook id acts as a bearer token, so keep it private.

### Environment variables

| Variable | Description |
|----------|-------------|
| `HOOKTAP_HOOK_ID` | Webhook id |
| `HOOKTAP_WEBHOOK_URL` | Full webhook URL (id is parsed out) |
| `HOOKTAP_BASE_URL` | Base URL override |

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Network or server error |
| `2` | Usage error (missing title, unknown type, no webhook configured) |
| `4` | Rate limited (HTTP 429 — max 1 request/second; safe to retry) |

## Examples

**Long-running command finished:**

```bash
make build && echo "build ok" | hooktap send "🏗️  $(basename $PWD)" \
  || echo "build failed" | hooktap send "💥 $(basename $PWD)" --type push
```

**Cron / monitoring:**

```bash
# /etc/cron.d/disk-check
0 * * * *  df -h / | tail -1 | hooktap send "Disk usage" --profile ops
```

**Pipe a server's JSON straight to your phone:**

```bash
curl -s https://api.example.com/status | hooktap send --raw
```

## Shell completions

```bash
# zsh
hooktap completion zsh > "${fpath[1]}/_hooktap"

# bash
hooktap completion bash > /usr/local/etc/bash_completion.d/hooktap

# fish
hooktap completion fish > ~/.config/fish/completions/hooktap.fish
```

Completions are context-aware: `--type` suggests the valid event types and `--profile` suggests your configured profiles.

## Related

- [HookTap Integrations](https://github.com/HookTap/hooktap-integrations) — recipes for cURL, CI, Python, Node, and more
- [HookTap Notify Action](https://github.com/HookTap/notify-action) — GitHub Action wrapper

> This repository is maintained by [github.com/HookTap](https://github.com/HookTap).

## License

MIT — see [LICENSE](./LICENSE).
