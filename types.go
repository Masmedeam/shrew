package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
)

type Message struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Rendered string `json:"-"`
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

type model struct {
	config    Config
	system    string
	history   []Message
	sessionID string
	input     textinput.Model
	viewport  viewport.Model
	spinner   spinner.Model
	renderer  *glamour.TermRenderer
	executing bool
	err       error
	width     int
	height    int
}

type Session struct {
	ID        string    `json:"id"`
	Timestamp string    `json:"timestamp"`
	Messages  []Message `json:"messages"`
}

type agentResponseMsg struct {
	content string
}

type chunkMsg struct {
	content string
	done    bool
}

type agentErrorMsg error
type commandOutputMsg string
