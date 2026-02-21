package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	systemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	userStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true)
	shrewStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	thinkStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#5C5C5C")).Padding(0, 1).Faint(true)
	execStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Italic(true)
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	inputStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#7D56F4")).Padding(0, 1)
)

func initialModel() *model {
	ti := textinput.New()
	ti.Placeholder = "Ask shrew..."
	ti.Focus()
	ti.CharLimit = 2000

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	// Glamour renderer - Using fixed "dark" style to prevent background querying leak
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(100),
	)

	return &model{
		input:    ti,
		spinner:  s,
		renderer: renderer,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(msg.Width, msg.Height-7) // Adjusted for header + footer
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(msg.Width-4),
		)
		// Clear cache to re-render with new width
		for i := range m.history {
			m.history[i].Rendered = ""
		}
		m.viewport.SetContent(m.renderHistory())
		m.input.Width = msg.Width - 8 // Account for border and padding
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+y":
			// Yank (copy) last assistant message
			for i := len(m.history) - 1; i >= 0; i-- {
				if m.history[i].Role == "assistant" {
					clipboard.WriteAll(m.history[i].Content)
					break
				}
			}
			return m, nil
		case "pgup", "pgdown", "up", "down":
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		case "enter":
			if m.executing || m.input.Value() == "" {
				return m, nil
			}

			userQuery := m.input.Value()
			m.history = append(m.history, Message{Role: "user", Content: userQuery})
			m.input.SetValue("")
			m.executing = true
			m.viewport.SetContent(m.renderHistory())
			m.viewport.GotoBottom()

			return m, tea.Batch(m.callAgentCmd(), m.spinner.Tick)
		}

	case tea.MouseMsg:
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, vpCmd

	case spinner.TickMsg:
		if m.executing {
			m.spinner, spCmd = m.spinner.Update(msg)
		}

	case agentResponseMsg:
		m.history = append(m.history, Message{Role: "assistant", Content: msg.content})
		updateSession(m.sessionID, m.history)
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()

		re := regexp.MustCompile(`(?s)<run>(.*?)</run>`)
		match := re.FindStringSubmatch(msg.content)
		if len(match) >= 2 {
			cmdStr := strings.TrimSpace(match[1])
			return m, m.runCommandCmd(cmdStr)
		}

		m.executing = false

	case commandOutputMsg:
		m.history = append(m.history, Message{Role: "user", Content: fmt.Sprintf("<output>\n%s\n</output>", string(msg))})
		updateSession(m.sessionID, m.history)
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
		return m, m.callAgentCmd()

	case agentErrorMsg:
		m.err = msg
		m.executing = false
		m.history = append(m.history, Message{Role: "system", Content: fmt.Sprintf("Error: %v", msg)})
		m.viewport.SetContent(m.renderHistory())
		m.viewport.GotoBottom()
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m *model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	header := titleStyle.Render("SHREW") + " " + systemStyle.Render(fmt.Sprintf("%s | %s | session: %s", m.config.Provider, m.config.Model, m.sessionID))

	indicator := "\n"
	if m.executing {
		indicator = "\n " + m.spinner.View() + " Thinking...\n"
	}

	footer := indicator + inputStyle.Width(m.width-4).Render(m.input.View())

	return header + "\n" + m.viewport.View() + footer
}

// --- Rendering Logic ---

func (m *model) renderHistory() string {
	var sb strings.Builder
	thinkRegex := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	runRegex := regexp.MustCompile(`(?s)<run>(.*?)</run>`)
	outputRegex := regexp.MustCompile(`(?s)<output>(.*?)</output>`)

	for i := range m.history {
		msg := &m.history[i]
		
		// Re-render if not cached or if it's the latest message (might be incomplete if streaming existed, but good practice)
		if msg.Rendered == "" {
			var msgSB strings.Builder
			switch msg.Role {
			case "user":
				if outputRegex.MatchString(msg.Content) {
					match := outputRegex.FindStringSubmatch(msg.Content)
					msgSB.WriteString(outputStyle.Render(fmt.Sprintf("[Output]:\n%s\n", match[1])))
				} else {
					msgSB.WriteString(userStyle.Render("\n> " + msg.Content + "\n"))
				}
			case "assistant":
				content := msg.Content
				content = thinkRegex.ReplaceAllStringFunc(content, func(match string) string {
					submatch := thinkRegex.FindStringSubmatch(match)
					msgSB.WriteString(thinkStyle.Render("Thinking:\n" + submatch[1]) + "\n")
					return ""
				})
				content = runRegex.ReplaceAllStringFunc(content, func(match string) string {
					submatch := runRegex.FindStringSubmatch(match)
					msgSB.WriteString(execStyle.Render(fmt.Sprintf("[Executing]: %s\n", submatch[1])))
					return ""
				})
				if strings.TrimSpace(content) != "" {
					renderedMarkdown, _ := m.renderer.Render(content)
					msgSB.WriteString(shrewStyle.Render("shrew: \n") + renderedMarkdown)
				}
			case "system":
				msgSB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(msg.Content) + "\n")
			}
			msg.Rendered = msgSB.String()
		}
		sb.WriteString(msg.Rendered)
	}
	return sb.String()
}

func (m *model) callAgentCmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := callAPI(m.config, m.system, m.history)
		if err != nil {
			return agentErrorMsg(err)
		}
		return agentResponseMsg{content: resp}
	}
}

func (m *model) runCommandCmd(cmdStr string) tea.Cmd {
	return func() tea.Msg {
		output, _ := executeCommand(cmdStr)
		return commandOutputMsg(output)
	}
}
