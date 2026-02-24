package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

//go:embed ui/index.html
var uiFS embed.FS

type Server struct {
	Engine *Engine
	mu     sync.Mutex
	subs   []chan Event
}

func NewServer(e *Engine) *Server {
	return &Server{Engine: e}
}

func (s *Server) Start(port int) error {
	http.HandleFunc("/", s.handleUI)
	http.HandleFunc("/events", s.handleEvents)
	http.HandleFunc("/chat", s.handleChat)

	fmt.Printf("Web UI available at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	f, err := uiFS.Open("ui/index.html")
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	io.Copy(w, f)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := s.Engine.Subscribe()
	
	for {
		select {
		case event := <-ch:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	go s.Engine.Process(req.Message)
	w.WriteHeader(http.StatusAccepted)
}
