package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
	"github.com/spf13/cobra"
)

type tuiScreen int

const (
	screenSend tuiScreen = iota
	screenProfiles
	screenSnippets
	screenDoctor
)

var (
	tuiChrome = lipgloss.NewStyle().Padding(1, 2)
	tuiTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	tuiMuted  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tuiActive = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	tuiBox    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(1, 2)
	tuiOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	tuiErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type tuiModel struct {
	screen    tuiScreen
	inputs    []textinput.Model
	focus     int
	eventType string
	status    string
	sending   bool
	width     int
	height    int
}

type sendDoneMsg struct {
	eventID string
	typ     string
	err     error
}

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive HookTap terminal UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd)
		},
	}
}

func runTUI(cmd *cobra.Command) error {
	p := tea.NewProgram(newTUIModel())
	_, err := p.Run()
	return err
}

func newTUIModel() tuiModel {
	labels := []string{"Title", "Body", "Hook override"}
	placeholders := []string{"Deploy finished", "staging is live", "optional webhook id or URL"}
	inputs := make([]textinput.Model, len(labels))
	for i := range inputs {
		ti := textinput.New()
		ti.Prompt = labels[i] + ": "
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 512
		ti.Width = 64
		if i == 0 {
			ti.Focus()
		}
		inputs[i] = ti
	}
	return tuiModel{
		screen:    screenSend,
		inputs:    inputs,
		eventType: client.DefaultType,
		status:    "Press Ctrl+S to send. Press 1-4 to switch screens.",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		for i := range m.inputs {
			m.inputs[i].Width = max(32, min(72, msg.Width-24))
		}
		return m, nil
	case sendDoneMsg:
		m.sending = false
		if msg.err != nil {
			m.status = tuiErr.Render("send failed: " + msg.err.Error())
		} else {
			m.status = tuiOK.Render(fmt.Sprintf("sent %s event %s", msg.typ, msg.eventID))
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "1":
			m.screen = screenSend
		case "2":
			m.screen = screenProfiles
		case "3":
			m.screen = screenSnippets
		case "4":
			m.screen = screenDoctor
		case "left":
			m.screen = (m.screen + 3) % 4
		case "right":
			m.screen = (m.screen + 1) % 4
		case "tab":
			if m.screen == screenSend {
				m.inputs[m.focus].Blur()
				m.focus = (m.focus + 1) % len(m.inputs)
				m.inputs[m.focus].Focus()
			}
		case "shift+tab":
			if m.screen == screenSend {
				m.inputs[m.focus].Blur()
				m.focus = (m.focus + len(m.inputs) - 1) % len(m.inputs)
				m.inputs[m.focus].Focus()
			}
		case "ctrl+s":
			if m.screen == screenSend && !m.sending {
				m.sending = true
				m.status = "sending..."
				return m, m.sendCmd()
			}
		case "ctrl+t":
			if m.screen == screenSend {
				m.eventType = nextEventType(m.eventType)
			}
		}
	}

	if m.screen == screenSend {
		var cmds []tea.Cmd
		for i := range m.inputs {
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m tuiModel) sendCmd() tea.Cmd {
	title := strings.TrimSpace(m.inputs[0].Value())
	body := strings.TrimSpace(m.inputs[1].Value())
	hook := strings.TrimSpace(m.inputs[2].Value())
	eventType := m.eventType

	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return sendDoneMsg{err: err}
		}
		prof := cfg.Profile(cfg.ResolveName(flagProfile))
		s, err := resolveSettings(flagURL, hook, eventType, prof)
		if err != nil {
			return sendDoneMsg{err: err}
		}
		if title == "" {
			return sendDoneMsg{err: fmt.Errorf("%w: title is required", errUsage)}
		}
		resp, err := client.New(s.baseURL).Send(context.Background(), s.hookID, client.Payload{
			Type:  eventType,
			Title: title,
			Body:  body,
		})
		if err != nil {
			return sendDoneMsg{err: err}
		}
		return sendDoneMsg{eventID: resp.EventID, typ: resp.Type}
	}
}

func (m tuiModel) View() string {
	var body string
	switch m.screen {
	case screenSend:
		body = m.sendView()
	case screenProfiles:
		body = profilesView()
	case screenSnippets:
		body = snippetsView()
	case screenDoctor:
		body = doctorView()
	}

	return tuiChrome.Render(strings.Join([]string{
		tuiTitle.Render("HookTap CLI"),
		m.tabsView(),
		tuiBox.Width(max(64, m.width-8)).Render(body),
		tuiMuted.Render("1 Send  2 Profiles  3 Snippets  4 Doctor  |  Ctrl+S send  Ctrl+T type  q quit"),
		m.status,
	}, "\n\n"))
}

func (m tuiModel) tabsView() string {
	names := []string{"Send", "Profiles", "Snippets", "Doctor"}
	parts := make([]string, len(names))
	for i, name := range names {
		label := fmt.Sprintf("%d %s", i+1, name)
		if tuiScreen(i) == m.screen {
			parts[i] = tuiActive.Render(label)
		} else {
			parts[i] = tuiMuted.Render(label)
		}
	}
	return strings.Join(parts, "   ")
}

func (m tuiModel) sendView() string {
	var b strings.Builder
	b.WriteString("Compose notification\n\n")
	for _, input := range m.inputs {
		b.WriteString(input.View())
		b.WriteByte('\n')
	}
	b.WriteString("\nType: ")
	b.WriteString(tuiActive.Render(m.eventType))
	b.WriteString("  ")
	b.WriteString(tuiMuted.Render("(Ctrl+T cycles push/feed/widget)"))
	if m.sending {
		b.WriteString("\n\nSending...")
	} else {
		b.WriteString("\n\nPress Ctrl+S to send.")
	}
	return b.String()
}

func profilesView() string {
	cfg, err := config.Load()
	if err != nil {
		return tuiErr.Render(err.Error())
	}
	names := cfg.ProfileNames()
	if len(names) == 0 {
		return "No profiles yet.\n\nRun hooktap setup or use this command:\n\n  hooktap config set hook_id YOUR_ID"
	}
	defaultName := cfg.ResolveName("")
	var b strings.Builder
	b.WriteString("Saved profiles\n\n")
	for _, name := range names {
		marker := " "
		if name == defaultName {
			marker = "*"
		}
		p := cfg.Profile(name)
		target := firstNonEmpty(p.HookID, p.URL)
		b.WriteString(fmt.Sprintf("%s %-16s %-8s %s\n", marker, name, firstNonEmpty(p.Type, client.DefaultType), target))
	}
	b.WriteString("\nUse: hooktap config use <profile>")
	return b.String()
}

func snippetsView() string {
	return strings.Join([]string{
		"Copy-ready snippets",
		"",
		"Send a simple event:",
		"  hooktap send \"Deploy finished\" --body \"staging is live\"",
		"",
		"Pipe command output:",
		"  make build 2>&1 | hooktap send \"Build output\"",
		"",
		"Notify when a command exits:",
		"  hooktap watch -- npm run build",
		"",
		"Raw JSON mapping:",
		"  curl -s https://api.example.com/status | hooktap send --raw",
	}, "\n")
}

func doctorView() string {
	path, pathErr := config.Path()
	cfg, cfgErr := config.Load()
	lines := []string{"Doctor", ""}
	if pathErr != nil {
		lines = append(lines, tuiErr.Render("config path: "+pathErr.Error()))
	} else {
		lines = append(lines, "config path: "+path)
	}
	if cfgErr != nil {
		lines = append(lines, tuiErr.Render("config load: "+cfgErr.Error()))
	} else {
		lines = append(lines, fmt.Sprintf("profiles: %d", len(cfg.Profiles)))
		lines = append(lines, "default: "+cfg.ResolveName(""))
	}
	lines = append(lines, "api health: run `hooktap ping`")
	return strings.Join(lines, "\n")
}

func nextEventType(current string) string {
	switch current {
	case client.TypePush:
		return client.TypeFeed
	case client.TypeFeed:
		return client.TypeWidget
	default:
		return client.TypePush
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
