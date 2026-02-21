package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

func callAPI(cfg Config, system string, history []Message) (string, error) {
	switch cfg.Provider {
	case "gemini":
		return callGemini(cfg, system, history)
	case "openai":
		return callOpenAI(cfg, system, history)
	case "ollama":
		return callOllama(cfg, system, history)
	case "cmd":
		return callExternalCmd(cfg, system, history)
	}
	return "", fmt.Errorf("unknown provider")
}

func callGemini(cfg Config, system string, history []Message) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", cfg.Model, cfg.GeminiKey)
	var contents []GeminiContent
	for _, m := range history {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, GeminiContent{Role: role, Parts: []GeminiPart{{Text: m.Content}}})
	}
	reqBody, _ := json.Marshal(GeminiRequest{
		SystemInstruction: &GeminiContent{Parts: []GeminiPart{{Text: system}}},
		Contents:          contents,
	})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
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
	if cfg.APIURL != "" {
		url = cfg.APIURL
	}
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)
	reqBody, _ := json.Marshal(map[string]interface{}{"model": cfg.Model, "messages": messages})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenAIKey)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("api error (%d): %s", resp.StatusCode, string(b))
	}
	var result struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response")
}

func callOllama(cfg Config, system string, history []Message) (string, error) {
	url := strings.TrimSuffix(cfg.OllamaURL, "/") + "/api/chat"
	messages := []Message{{Role: "system", Content: system}}
	messages = append(messages, history...)
	reqBody, _ := json.Marshal(map[string]interface{}{"model": cfg.Model, "messages": messages, "stream": false})
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Message Message `json:"message"`
	}
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
	if err != nil {
		return "", fmt.Errorf("bridge error: %v, stderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(out.String()), nil
}
