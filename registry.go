package main

type ModelProvider struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Endpoint string   `json:"endpoint"`
	Models   []string `json:"models"`
}

var ModelRegistry = []ModelProvider{
	{
		ID:       "gemini",
		Name:     "Google Gemini",
		Endpoint: "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		Models: []string{
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite-preview-02-05",
			"gemini-2.0-pro-exp-02-05",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
	},
	{
		ID:       "openai",
		Name:     "OpenAI",
		Endpoint: "https://api.openai.com/v1/chat/completions",
		Models: []string{
			"gpt-4o",
			"gpt-4o-mini",
			"o1",
			"o1-mini",
			"o3-mini",
		},
	},
	{
		ID:       "anthropic",
		Name:     "Anthropic Claude",
		Endpoint: "https://api.anthropic.com/v1/messages",
		Models: []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
			"claude-3-opus-20240229",
		},
	},
	{
		ID:       "deepseek",
		Name:     "DeepSeek",
		Endpoint: "https://api.deepseek.com/chat/completions",
		Models: []string{
			"deepseek-chat",
			"deepseek-reasoner",
		},
	},
	{
		ID:       "groq",
		Name:     "Groq",
		Endpoint: "https://api.groq.com/openai/v1/chat/completions",
		Models: []string{
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
		},
	},
	{
		ID:       "mistral",
		Name:     "Mistral AI",
		Endpoint: "https://api.mistral.ai/v1/chat/completions",
		Models: []string{
			"mistral-large-latest",
			"mistral-medium-latest",
			"mistral-small-latest",
			"codestral-latest",
		},
	},
	{
		ID:       "ollama",
		Name:     "Ollama (Local)",
		Endpoint: "http://localhost:11434/api/chat",
		Models: []string{
			"llama3.1",
			"llama3.3",
			"mistral",
			"mixtral",
			"qwen2.5-coder",
		},
	},
}
