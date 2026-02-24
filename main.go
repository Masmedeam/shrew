package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const baseSystemPrompt = `You are "shrew", a minimalist AI agent.
CRITICAL: Do not use Markdown (no bold, no italics, no markdown lists, no backticks for code).
Respond in PLAIN TEXT only.
To execute shell commands, wrap them in <run>tags: <run>ls -la</run>.
To reason, use <think>...</think> tags.
Use standard CLI tools.`

func main() {
	listFlag := flag.Bool("list", false, "List all available sessions")
	portFlag := flag.Int("port", 8080, "Port for the Web UI")
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
		APIURL:    os.Getenv("SHREW_API_URL"),
		CustomCmd: os.Getenv("SHREW_COMMAND"),
	}

	modelEnv := os.Getenv("SHREW_MODEL")
	if modelEnv == "" {
		modelEnv = "ollama/qwen2.5-coder:7b"
	}

	parts := strings.SplitN(modelEnv, "/", 2)
	if len(parts) != 2 {
		fmt.Println("Invalid SHREW_MODEL. Expected 'provider/model'")
		os.Exit(1)
	}
	cfg.Provider = parts[0]
	cfg.Model = parts[1]

	sessionID := time.Now().Format("2006-01-02-15-04-05")
	history := []Message{{Role: "user", Content: "Context: " + gatherContext()}}
	
	engine := NewEngine(cfg, baseSystemPrompt+loadSkills(), sessionID, history)
	server := NewServer(engine)

	// Start Server
	go func() {
		if err := server.Start(*portFlag); err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Terminal REPL
	fmt.Printf("\nShrew is active.\n")
	fmt.Printf("   Web UI: http://localhost:%d\n", *portFlag)
	fmt.Printf("   Terminal: Type below and press Enter\n\n")

	// Subscribe terminal to engine events
	events := engine.Subscribe()
	go func() {
		for event := range events {
			switch event.Type {
			case EventThinking:
				fmt.Print("...thinking")
			case EventExecuting:
				fmt.Printf("\n> [run]: %s\n", event.Content)
			case EventOutput:
				fmt.Printf("[output]: %s\n", event.Content)
			case EventResponse:
				fmt.Printf("\nshrew: %s\n\n", event.Content)
			case EventError:
				fmt.Printf("\nError: %s\n", event.Content)
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}
		engine.Process(input)
	}
}
