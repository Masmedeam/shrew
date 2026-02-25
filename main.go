package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"
)

const baseSystemPrompt = `You are "shrew", a minimalist AI agent.
CRITICAL: Do not use Markdown (no bold, no italics, no markdown lists, no backticks for code).
Respond in PLAIN TEXT only.
To execute shell commands, wrap them in <run>tags: <run>ls -la</run>.
To reason, use <think>...</think> tags.
Use standard CLI tools.

VAULT:
To use secrets (bearer tokens, API keys) in shell commands without seeing them, use the placeholder [[vault:NAME]] inside <run> tags.
Example: <run>curl -H "Authorization: Bearer [[vault:OPENAI_API_KEY]]" ...</run>
The system will automatically inject the secret before execution.
To store a secret: <vault_set key="NAME" value="SECRET_VALUE"/>.
To see which keys are available in the vault without seeing their values: <vault_list/>.
If you need a secret but don't know the key name, use <vault_list/> first to help the user.
Do not use <vault_get> if you only need the secret for a command.
If you need to see a secret for other reasons, use <vault_get key="NAME"/>.

SKILLS:
If you need documentation for an API, first check if you have it using <get_skill name="service_name"/>.
If not found, search for it (e.g., using curl) or ask the user.
To save documentation for future use, use <save_skill name="service_name">DOCS_CONTENT</save_skill>.`

func main() {
	listFlag := flag.Bool("list", false, "List all available sessions")
	portFlag := flag.Int("port", 8080, "Port for the Web UI")
	flag.Parse()

	if *listFlag {
		db, err := InitDB("shrew.db")
		if err != nil {
			fmt.Printf("Error initializing DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		sessions, _ := db.ListSessions()
		fmt.Println("Available Sessions:")
		for _, s := range sessions {
			fmt.Printf("- %s (Last update: %s)\n", s.ID, s.Timestamp)
		}
		return
	}

	loadEnv()
	db, err := InitDB("shrew.db")
	if err != nil {
		fmt.Printf("Error initializing DB: %v\n", err)
		os.Exit(1)
	}
	// Note: In a real app, you might want to handle closing db more gracefully
	// but for a CLI tool this is often acceptable until shutdown.

	cfg := Config{
		APIKey:             os.Getenv("SHREW_API_KEY"),
		APIURL:             os.Getenv("SHREW_API_URL"),
		Model:              os.Getenv("SHREW_MODEL"),
		CustomInstructions: os.Getenv("SHREW_CUSTOM_INSTRUCTIONS"),
	}

	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.openai.com/v1/chat/completions"
	}

	sessionID := time.Now().Format("2006-01-02-15-04-05")
	history := []Message{{Role: "user", Content: "Context: " + gatherContext()}}
	
	engine := NewEngine(cfg, baseSystemPrompt, sessionID, history, db)
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
