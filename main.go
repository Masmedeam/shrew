package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const baseSystemPrompt = `You are "shrew", a minimalist CLI coding agent.
You have the power to execute shell commands on the user's machine.
To execute a command, wrap it in <run>tags like this: <run>ls -la</run>.
To reason about a problem, use <think>...</think> tags.
Use standard CLI tools. Always explain what you are doing briefly.
After running a command, you will receive the output.
Continue until the task is complete.
Your output should be formatted as Markdown.`

func main() {
	sessionID := flag.String("session", "", "Load a specific session by ID")
	listFlag := flag.Bool("list", false, "List all available sessions")
	flag.Parse()

	if *listFlag {
		h, _ := loadHistory()
		fmt.Println("Available Sessions:")
		for id, s := range h.Sessions {
			fmt.Printf("- %s (Last update: %s)\n", id, s.Timestamp)
		}
		return
	}

	loadEnv()
	cfg := Config{
		GeminiKey: os.Getenv("GEMINI_API_KEY"),
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
		OllamaURL: os.Getenv("OLLAMA_URL"),
		Provider:  os.Getenv("SHREW_PROVIDER"),
		Model:     os.Getenv("SHREW_MODEL"),
		APIURL:    os.Getenv("SHREW_API_URL"),
		CustomCmd: os.Getenv("SHREW_COMMAND"),
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

	if *sessionID != "" {
		messages, err := getSession(*sessionID)
		if err != nil {
			fmt.Printf("Error loading session: %v\n", err)
			os.Exit(1)
		}
		if messages != nil {
			m.history = messages
			m.sessionID = *sessionID
		} else {
			fmt.Printf("Session %s not found. Starting new session.\n", *sessionID)
			m.sessionID = *sessionID
			m.history = append(m.history, Message{Role: "user", Content: "Context: " + gatherContext()})
		}
	} else {
		m.sessionID = time.Now().Format("2006-01-02-15-04-05")
		m.history = append(m.history, Message{Role: "user", Content: "Context: " + gatherContext()})
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
