package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type EventType string

const (
	EventThinking     EventType = "thinking"
	EventExecuting    EventType = "executing"
	EventFileOp       EventType = "file_op"
	EventOutput       EventType = "output"
	EventResponse     EventType = "response"
	EventError        EventType = "error"
	EventUserMessage  EventType = "user_message"
)

type Event struct {
	Type    EventType `json:"type"`
	Content string    `json:"content"`
}

type Engine struct {
	Config      Config
	System      string
	History     []Message
	SessionID   string
	Subscribers []chan Event
	DB          *DB
	mu          sync.Mutex
}

func NewEngine(cfg Config, system string, sessionID string, history []Message, db *DB) *Engine {
	return &Engine{
		Config:    cfg,
		System:    system,
		History:   history,
		SessionID: sessionID,
		DB:        db,
	}
}

func (e *Engine) Subscribe() chan Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan Event, 10)
	e.Subscribers = append(e.Subscribers, ch)
	return ch
}

func (e *Engine) broadcast(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, sub := range e.Subscribers {
		select {
		case sub <- event:
		default:
		}
	}
}

func (e *Engine) Process(input string) {
	e.mu.Lock()
	e.History = append(e.History, Message{Role: "user", Content: input})
	e.mu.Unlock()
	e.broadcast(Event{Type: EventUserMessage, Content: input})
	e.runLoop()
}

func (e *Engine) runLoop() {
	for {
		e.broadcast(Event{Type: EventThinking, Content: ""})
		resp, err := callAPI(e.Config, e.System, e.History)
		if err != nil {
			e.broadcast(Event{Type: EventError, Content: err.Error()})
			return
		}

		e.mu.Lock()
		e.History = append(e.History, Message{Role: "assistant", Content: resp})
		e.DB.SaveSession(Session{ID: e.SessionID, Messages: e.History, Timestamp: time.Now().Format(time.RFC3339)})
		e.mu.Unlock()
		e.broadcast(Event{Type: EventResponse, Content: resp})

		// Multi-tag extraction
		handled := e.handleTags(resp)
		if !handled {
			break
		}
	}
}

func (e *Engine) handleTags(content string) bool {
	// 1. Check for <run>
	runRe := regexp.MustCompile(`(?s)<run>(.*?)</run>`)
	if match := runRe.FindStringSubmatch(content); len(match) >= 2 {
		cmdStr := strings.TrimSpace(match[1])
		e.broadcast(Event{Type: EventExecuting, Content: cmdStr})
		output, _ := executeCommand(cmdStr)
		e.addOutput(fmt.Sprintf("<output>\n%s\n</output>", output), output)
		return true
	}

	// 2. Check for <read>
	readRe := regexp.MustCompile(`(?s)<read>(.*?)</read>`)
	if match := readRe.FindStringSubmatch(content); len(match) >= 2 {
		path := strings.TrimSpace(match[1])
		e.broadcast(Event{Type: EventFileOp, Content: "Reading " + path})
		data, err := os.ReadFile(path)
		output := string(data)
		if err != nil {
			output = fmt.Sprintf("Error reading file: %v", err)
		}
		e.addOutput(fmt.Sprintf("<output>\n%s\n</output>", output), output)
		return true
	}

	// 3. Check for <write>
	writeRe := regexp.MustCompile(`(?s)<write>(.*?)</write>(.*?)[\n\r]*</write>`)
	if match := writeRe.FindStringSubmatch(content); len(match) >= 3 {
		path := strings.TrimSpace(match[1])
		body := match[2]
		e.broadcast(Event{Type: EventFileOp, Content: "Writing " + path})
		err := os.WriteFile(path, []byte(body), 0644)
		output := "File written successfully"
		if err != nil {
			output = fmt.Sprintf("Error writing file: %v", err)
		}
		e.addOutput(fmt.Sprintf("<output>\n%s\n</output>", output), output)
		return true
	}

	// 4. Check for <vault_get>
	vaultRe := regexp.MustCompile(`<vault_get\s+key="(.*?)"\s*/>`)
	if match := vaultRe.FindStringSubmatch(content); len(match) >= 2 {
		key := match[1]
		val, err := e.DB.GetSecret(key)
		output := val
		if err != nil {
			output = fmt.Sprintf("Error: Secret '%s' not found in vault.", key)
		}
		e.addOutput(fmt.Sprintf("<vault_output key=\"%s\">\n%s\n</vault_output>", key, output), "Retrieved secret from vault: "+key)
		return true
	}

	// 5. Check for <save_skill>
	skillSaveRe := regexp.MustCompile(`(?s)<save_skill\s+name="(.*?)">(.*?)</save_skill>`)
	if match := skillSaveRe.FindStringSubmatch(content); len(match) >= 3 {
		name := match[1]
		docs := match[2]
		e.DB.SaveSkill(name, docs)
		e.addOutput(fmt.Sprintf("Skill '%s' saved successfully.", name), "Learned new skill: "+name)
		return true
	}

	// 6. Check for <get_skill>
	skillGetRe := regexp.MustCompile(`<get_skill\s+name="(.*?)"\s*/>`)
	if match := skillGetRe.FindStringSubmatch(content); len(match) >= 2 {
		name := match[1]
		docs, err := e.DB.GetSkill(name)
		output := docs
		if err != nil {
			output = fmt.Sprintf("Error: Skill '%s' not found.", name)
		}
		e.addOutput(fmt.Sprintf("<skill_output name=\"%s\">\n%s\n</skill_output>", name, output), "Retrieved skill docs: "+name)
		return true
	}

	return false
}

func (e *Engine) addOutput(fullMsg string, display string) {
	e.mu.Lock()
	e.History = append(e.History, Message{Role: "user", Content: fullMsg})
	e.DB.SaveSession(Session{ID: e.SessionID, Messages: e.History, Timestamp: time.Now().Format(time.RFC3339)})
	e.mu.Unlock()
	e.broadcast(Event{Type: EventOutput, Content: display})
}
