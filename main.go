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
	APIURL     string // For generic OpenAI-compatible providers
	CustomCmd  string // For the "cmd" provider bridge
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

// --- Constants & Prompts ---

const baseSystemPrompt = `You are "shrew", a minimalist CLI coding agent.
You have the power to execute shell commands on the user's machine.
To execute a command, wrap it in <run>tags like this: <run>ls -la</run>
You should use standard CLI tools (cat, echo, grep, sed, git, go, etc.) to read, write, and manage files.
If a tool is missing, you can try to install it.
Always explain what you are doing briefly.
After running a command, you will receive the output.
Continue until the task is complete.
Keep your responses concise.
`

// --- Utilities ---

func loadEnv() {
	f, err := os.Open(".env")
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			val := strings.Trim(parts[1], `"'`)
			os.Setenv(parts[0], val)
		}
	}
}

func loadSkills() string {
	var skills strings.Builder
	skills.WriteString("\n--- Specialized Skills ---\n")
	files, _ := os.ReadDir("skills")
	count := 0
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			content, err := os.ReadFile(filepath.Join("skills", file.Name()))
			if err == nil {
				skills.WriteString(fmt.Sprintf("### Skill: %s\n%s\n\n", file.Name(), string(content)))
				count++
			}
		}
	}
	if count == 0 { return "" }
	return skills.String()
}

func gatherContext() string {
	var files []string
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(path, ".") || strings.Contains(path, "node_modules") {
			return nil
		}
		if len(files) < 20 { files = append(files, path) }
		return nil
	})
	wd, _ := os.Getwd()
	return fmt.Sprintf("Working Dir: %s\nFiles: %s", wd, strings.Join(files, ", "))
}

// --- Execution Loop ---

func main() {
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

	// Dynamic Defaults
	if cfg.Provider == "" {
		if cfg.GeminiKey != "" { cfg.Provider = "gemini"
		} else if cfg.OpenAIKey != "" { cfg.Provider = "openai"
		} else { cfg.Provider = "ollama" }
	}

	if cfg.Model == "" {
		switch cfg.Provider {
		case "gemini": cfg.Model = "gemini-3-flash-preview"
		case "openai": cfg.Model = "gpt-4o"
		case "ollama": cfg.Model = "qwen2.5-coder:7b"
		}
	}

	skillsPrompt := loadSkills()
	fullSystemPrompt := baseSystemPrompt + skillsPrompt

	fmt.Printf("shrew: Minimalist Agent | Provider: %s | Model: %s\n", cfg.Provider, cfg.Model)
	if cfg.Provider == "cmd" { fmt.Printf("Using Custom Command Bridge: %s\n", cfg.CustomCmd) }
	fmt.Println("---------------------------------------------------------")

	history := []Message{}
	history = append(history, Message{Role: "user", Content: "Context: " + gatherContext()})

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "exit" || input == "quit" { break }
		if input == "" { continue }

		history = append(history, Message{Role: "user", Content: input})

		for {
			response, err := callAPI(cfg, fullSystemPrompt, history)
			if err != nil {
				fmt.Printf("API Error: %v\n", err)
				break
			}

			fmt.Println(response)
			history = append(history, Message{Role: "assistant", Content: response})

			re := regexp.MustCompile(`(?s)<run>(.*?)</run>`)
			match := re.FindStringSubmatch(response)
			if len(match) < 2 { break }

			cmdStr := strings.TrimSpace(match[1])
			fmt.Printf("\n[Executing]: %s\n", cmdStr)
			output, err := executeCommand(cmdStr)
			if err != nil { output = fmt.Sprintf("Error: %v\nOutput: %s", err, output) }
			if output == "" { output = "(no output)" }
			fmt.Printf("[Output]: %s\n", output)
			history = append(history, Message{Role: "user", Content: "Command Output:\n" + output})
		}
	}
}

func executeCommand(cmdStr string) (string, error) {
	cmd := exec.Command("bash", "-c", cmdStr)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	return out.String() + stderr.String(), err
}

// --- API Calls ---

func callAPI(cfg Config, system string, history []Message) (string, error) {
	switch cfg.Provider {
	case "gemini": return callGemini(cfg, system, history)
	case "openai": return callOpenAI(cfg, system, history)
	case "ollama": return callOllama(cfg, system, history)
	case "cmd":    return callExternalCmd(cfg, system, history)
	}
	return "", fmt.Errorf("unknown provider: %s", cfg.Provider)
}

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
	return "", fmt.Errorf("no response from gemini")
}

func callOpenAI(cfg Config, system string, history []Message) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	if cfg.APIURL != "" { url = cfg.APIURL }
	
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct { Message Message `json:"message"` } `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Choices) > 0 { return result.Choices[0].Message.Content, nil }
	return "", fmt.Errorf("no response from openai")
}

func callOllama(cfg Config, system string, history []Message) (string, error) {
	url := cfg.OllamaURL + "/api/chat"
	if strings.Contains(url, "localhost") && !strings.HasSuffix(url, "/api/chat") {
		url = strings.TrimSuffix(url, "/") + "/api/chat"
	}
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)
	reqBody, _ := json.Marshal(map[string]interface{}{ "model": cfg.Model, "messages": messages, "stream": false })
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil { return "", err }
	defer resp.Body.Close()
	var result struct { Message Message `json:"message"` }
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Message.Content, nil
}

func callExternalCmd(cfg Config, system string, history []Message) (string, error) {
	if cfg.CustomCmd == "" { return "", fmt.Errorf("SHREW_COMMAND is not set") }
	
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
