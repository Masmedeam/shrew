package main

import (
	"encoding/json"
	"os"
	"time"
)

const historyFile = "history.json"

type History struct {
	Sessions map[string]Session `json:"sessions"`
}

func loadHistory() (History, error) {
	var h History
	h.Sessions = make(map[string]Session)

	data, err := os.ReadFile(historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return h, err
	}

	err = json.Unmarshal(data, &h)
	return h, err
}

func saveHistory(h History) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyFile, data, 0644)
}

func getSession(id string) ([]Message, error) {
	h, err := loadHistory()
	if err != nil {
		return nil, err
	}
	if s, ok := h.Sessions[id]; ok {
		return s.Messages, nil
	}
	return nil, nil
}

func updateSession(id string, messages []Message) error {
	h, err := loadHistory()
	if err != nil {
		return err
	}

	h.Sessions[id] = Session{
		ID:        id,
		Timestamp: time.Now().Format(time.RFC3339),
		Messages:  messages,
	}

	return saveHistory(h)
}
