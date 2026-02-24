package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
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
	http.HandleFunc("/sessions", s.handleListSessions)
	http.HandleFunc("/session", s.handleGetSession)
	http.HandleFunc("/session/new", s.handleNewSession)
	http.HandleFunc("/vault", s.handleVault)
	http.HandleFunc("/skills", s.handleSkills)

	fmt.Printf("Web UI available at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.Engine.DB.ListSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	sess, err := s.Engine.DB.GetSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.Engine.mu.Lock()
	s.Engine.SessionID = sess.ID
	s.Engine.History = sess.Messages
	s.Engine.mu.Unlock()
	json.NewEncoder(w).Encode(sess)
}

func (s *Server) handleNewSession(w http.ResponseWriter, r *http.Request) {
	s.Engine.mu.Lock()
	s.Engine.SessionID = time.Now().Format("2006-01-02-15-04-05")
	s.Engine.History = []Message{{Role: "user", Content: "Context: " + gatherContext()}}
	s.Engine.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": s.Engine.SessionID})
}

func (s *Server) handleVault(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		secrets, _ := s.Engine.DB.ListSecrets()
		json.NewEncoder(w).Encode(secrets)
	case http.MethodPost:
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		s.Engine.DB.SaveSecret(req.Key, req.Value)
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		key := r.URL.Query().Get("key")
		s.Engine.DB.DeleteSecret(key)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	skills, _ := s.Engine.DB.ListSkills()
	json.NewEncoder(w).Encode(skills)
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
