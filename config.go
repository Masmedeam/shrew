package main

import (
	"encoding/json"
	"os"
)

type ConfigFile struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIURL    string `json:"api_url"`
	GeminiKey string `json:"gemini_api_key"`
	OpenAIKey string `json:"openai_api_key"`
	OllamaURL string `json:"ollama_url"`
	CustomCmd string `json:"custom_command"`
}

func loadConfigFile(path string) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *ConfigFile) toConfig() Config {
	return Config{
		Provider:  c.Provider,
		Model:     c.Model,
		APIURL:    c.APIURL,
		GeminiKey: c.GeminiKey,
		OpenAIKey: c.OpenAIKey,
		OllamaURL: c.OllamaURL,
		CustomCmd: c.CustomCmd,
	}
}

// Config precedence: CLI flags > env vars > config file > defaults
func mergeConfig(cfg Config, configFile *ConfigFile) Config {
	if configFile != nil {
		fileCfg := configFile.toConfig()
		if cfg.Provider == "" {
			cfg.Provider = fileCfg.Provider
		}
		if cfg.Model == "" {
			cfg.Model = fileCfg.Model
		}
		if cfg.APIURL == "" {
			cfg.APIURL = fileCfg.APIURL
		}
		if cfg.GeminiKey == "" {
			cfg.GeminiKey = fileCfg.GeminiKey
		}
		if cfg.OpenAIKey == "" {
			cfg.OpenAIKey = fileCfg.OpenAIKey
		}
		if cfg.OllamaURL == "" {
			cfg.OllamaURL = fileCfg.OllamaURL
		}
		if cfg.CustomCmd == "" {
			cfg.CustomCmd = fileCfg.CustomCmd
		}
	}
	return cfg
}
