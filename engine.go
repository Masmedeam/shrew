package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
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
	mu          sync.Mutex
}

func NewEngine(cfg Config, system string, sessionID string, history []Message) *Engine {
	return &Engine{
		Config:    cfg,
		System:    system,
		History:   history,
		SessionID: sessionID,
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
		updateSession(e.SessionID, e.History)
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

	return false
}

func (e *Engine) addOutput(fullMsg string, display string) {
	e.mu.Lock()
	e.History = append(e.History, Message{Role: "user", Content: fullMsg})
	updateSession(e.SessionID, e.History)
	e.mu.Unlock()
	e.broadcast(Event{Type: EventOutput, Content: display})
}
