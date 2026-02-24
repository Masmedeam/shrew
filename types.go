package main

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Config struct {
	GeminiKey string
	OpenAIKey string
	OllamaURL string
	Model     string
	Provider  string
	APIURL    string
	CustomCmd string
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

type Session struct {
	ID        string    `json:"id"`
	Timestamp string    `json:"timestamp"`
	Messages  []Message `json:"messages"`
}

type Skill struct {
	Name          string `json:"name"`
	Documentation string `json:"documentation"`
}
