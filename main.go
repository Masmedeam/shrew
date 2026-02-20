package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// --- Types ---

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Config struct {
	GeminiKey  string
	OpenAIKey  string
	OllamaURL  string
	Model      string
	Provider   string
	APIURL     string
	CustomCmd  string
}

type GeminiRequest struct {
	SystemInstruction *GeminiContent  `json:"system_instruction,omitempty"`
	Contents          []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}


// --- TUI Model ---

type model struct {
	config    Config
	system    string
	history   []Message
	input     textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model
	renderer  *glamour.TermRenderer
	executing bool
	err       error
	width     int
	height    int
}

type agentResponseMsg struct {
	content string
}
type agentErrorMsg error
type commandOutputMsg string

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	systemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	userStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ADD8")).Bold(true)
	shrewStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	thinkStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#5C5C5C")).Padding(0, 1).Faint(true)
	execStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Italic(true)
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
)


func initialModel() *model {
	ti := textinput.New()
	ti.Placeholder = "Ask shrew..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Glamour renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
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
		m.viewport = viewport.New(msg.Width, msg.Height-4)
		m.renderer, _ = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(msg.Width-4),
		)
		m.viewport.SetContent(m.renderHistory())
		m.input.Width = msg.Width - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
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

			return m, m.callAgentCmd()
		}

	case agentResponseMsg:
		m.history = append(m.history, Message{Role: "assistant", Content: msg.content})
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
	if m.executing {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m *model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	header := titleStyle.Render("SHREW") + " " + systemStyle.Render(fmt.Sprintf("%s | %s", m.config.Provider, m.config.Model))
	
	var indicator string
	if m.executing {
		indicator = " " + m.spinner.View() + " Thinking..."
	}
	
	footer := "\n" + m.input.View() + indicator

	return header + "\n" + m.viewport.View() + footer
}

// --- Rendering Logic ---

func (m *model) renderHistory() string {
	var sb strings.Builder
	thinkRegex := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	runRegex := regexp.MustCompile(`(?s)<run>(.*?)</run>`)
	outputRegex := regexp.MustCompile(`(?s)<output>(.*?)</output>`)

	for _, msg := range m.history {
		switch msg.Role {
		case "user":
			if outputRegex.MatchString(msg.Content) {
				match := outputRegex.FindStringSubmatch(msg.Content)
				sb.WriteString(outputStyle.Render(fmt.Sprintf("[Output]:\n%s\n", match[1])))
			} else {
				sb.WriteString(userStyle.Render("\n> " + msg.Content + "\n"))
			}
		case "assistant":
			content := msg.Content
			content = thinkRegex.ReplaceAllStringFunc(content, func(match string) string {
				submatch := thinkRegex.FindStringSubmatch(match)
				sb.WriteString(thinkStyle.Render("Thinking:\n" + submatch[1]) + "\n")
				return ""
			})
			content = runRegex.ReplaceAllStringFunc(content, func(match string) string {
				submatch := runRegex.FindStringSubmatch(match)
				sb.WriteString(execStyle.Render(fmt.Sprintf("[Executing]: %s\n", submatch[1])))
				return ""
			})
			if strings.TrimSpace(content) != "" {
				renderedMarkdown, _ := m.renderer.Render(content)
				sb.WriteString(shrewStyle.Render("shrew: \n") + renderedMarkdown)
			}
		case "system":
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(msg.Content) + "\n")
		}
	}
	return sb.String()
}


// --- Commands ---

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


// --- Main & Logic ---
func main() {
	// f, _ := tea.LogToFile("shrew_debug.log", "debug")
	// defer f.Close()

	loadEnv()
	cfg := Config{
		GeminiKey:  os.Getenv("GEMINI_API_KEY"),
		OpenAIKey:  os.Getenv("OPENAI_API_KEY"),
		OllamaURL:  os.Getenv("OLLAMA_URL"),
		Provider:   os.Getenv("SHREW_PROVIDER"),
		Model:      os.Getenv("SHREW_MODEL"),
		APIURL:     os.Getenv("SHREW_API_URL"),
		CustomCmd:  os.Getenv("SHREW_COMMAND"),
	}

	// Defaults
	if cfg.Provider == "" {
		if cfg.GeminiKey != "" {
			cfg.Provider = "gemini"
		} else if cfg.OpenAIKey != "" {
			cfg.Provider = "openai"
		} else {
			cfg.Provider = "ollama"
		}
	}
	if cfg.Model == "" {
		switch cfg.Provider {
		case "gemini":
			cfg.Model = "gemini-3-flash-preview"
		case "openai":
			cfg.Model = "gpt-4o"
		case "ollama":
			cfg.Model = "qwen2.5-coder:7b"
		}
	}

	m := initialModel()
	m.config = cfg
	m.system = baseSystemPrompt + loadSkills()
	m.history = append(m.history, Message{Role: "user", Content: "Context: " + gatherContext()})

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

const baseSystemPrompt = `You are "shrew", a minimalist CLI coding agent.
You have the power to execute shell commands on the user's machine.
To execute a command, wrap it in <run>tags like this: <run>ls -la</run>.
To reason about a problem, use <think>...</think> tags.
Use standard CLI tools. Always explain what you are doing briefly.
After running a command, you will receive the output.
Continue until the task is complete.
Your output should be formatted as Markdown.`

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil { return }
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			val := strings.Trim(parts[1], `"'`)
			os.Setenv(parts[0], val)
		}
	}
}

func loadSkills() string {
	var skills strings.Builder
	files, _ := os.ReadDir("skills")
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			content, _ := os.ReadFile(filepath.Join("skills", file.Name()))
			skills.WriteString(fmt.Sprintf("\n### Skill: %s\n%s\n", file.Name(), string(content)))
		}
	}
	return skills.String()
}

func gatherContext() string {
	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") { return nil }
		if len(files) < 15 { files = append(files, path) }
		return nil
	})
	wd, _ := os.Getwd()
	return fmt.Sprintf("Working Dir: %s\nFiles: %s", wd, strings.Join(files, ", "))
}

func executeCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(out.String() + stderr.String()), err
}

func callAPI(cfg Config, system string, history []Message) (string, error) {
	switch cfg.Provider {
	case "gemini": return callGemini(cfg, system, history)
	case "openai": return callOpenAI(cfg, system, history)
	case "ollama": return callOllama(cfg, system, history)
	case "cmd":    return callExternalCmd(cfg, system, history)
	}
	return "", fmt.Errorf("unknown provider")
}

// ... (API call functions are the same) ...
func callGemini(cfg Config, system string, history []Message) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", cfg.Model, cfg.GeminiKey)
	var contents []GeminiContent
	for _, m := range history {
		role := m.Role
		if role == "assistant" { role = "model" }
		contents = append(contents, GeminiContent{Role: role, Parts: []GeminiPart{{Text: m.Content}}})
	}
	reqBody, _ := json.Marshal(GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: system}}},
		Contents:          contents,
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return "", err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini error (%d): %s", resp.StatusCode, string(b))
	}
	var gResp GeminiResponse
	json.NewDecoder(resp.Body).Decode(&gResp)
	if len(gResp.Candidates) > 0 && len(gResp.Candidates[0].Content.Parts) > 0 {
		return gResp.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("no response")
}

func callOpenAI(cfg Config, system string, history []Message) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	if cfg.APIURL != "" { url = cfg.APIURL }
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)
	reqBody, _ := json.Marshal(map[string]interface{}{"model": cfg.Model, "messages": messages})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)
	resp, err := (&http.Client{}).Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error (%d): %s", resp.StatusCode, string(b))
	}
	var result struct {
		Choices []struct { Message Message `json:"message" ` } `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Choices) > 0 { return result.Choices[0].Message.Content, nil }
	return "", fmt.Errorf("no response")
}

func callOllama(cfg Config, system string, history []Message) (string, error) {
	url := strings.TrimSuffix(cfg.OllamaURL, "/") + "/api/chat"
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)
	reqBody, _ := json.Marshal(map[string]interface{}{"model": cfg.Model, "messages": messages, "stream": false})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result struct { Message Message `json:"message"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Message.Content, nil
}

func callExternalCmd(cfg Config, system string, history []Message) (string, error) {
	fullPrompt := []Message{{Role: "system", Content: system}}
	fullPrompt = append(fullPrompt, history...)
	jsonData, _ := json.Marshal(fullPrompt)
	cmd := exec.Command("bash", "-c", cfg.CustomCmd)
	cmd.Stdin = bytes.NewReader(jsonData)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil { return "", fmt.Errorf("bridge error: %v, stderr: %s", err, stderr.String()) }
	return strings.TrimSpace(out.String()), nil
}
