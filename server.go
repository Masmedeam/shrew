package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed ui/*
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
	http.HandleFunc("/session", s.handleSessionRoute)
	http.HandleFunc("/session/new", s.handleNewSession)
	http.HandleFunc("/vault", s.handleVault)
	http.HandleFunc("/skills", s.handleSkills)

	fmt.Printf("Web UI available at http://localhost:%d\n", port)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *Server) handleSessionRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetSession(w, r)
	case http.MethodDelete:
		s.handleDeleteSession(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}
	err := s.Engine.DB.DeleteSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.Engine.DB.ListSessions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sess)
}

func (s *Server) handleNewSession(w http.ResponseWriter, r *http.Request) {
	s.Engine.mu.Lock()
	s.Engine.SessionID = time.Now().Format("2006-01-02-15-04-05")
	s.Engine.History = []Message{{Role: "user", Content: "Context: " + gatherContext()}}
	s.Engine.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": s.Engine.SessionID})
}

func (s *Server) handleVault(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		secrets, _ := s.Engine.DB.ListSecrets()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(secrets)
	case http.MethodPost:
		var req struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		s.Engine.DB.SaveSecret(req.Key, req.Value)

		// Sync with .env and engine if it's a config key
		configKeys := map[string]bool{
			"SHREW_API_KEY":             true,
			"SHREW_API_URL":             true,
			"SHREW_MODEL":               true,
			"SHREW_CUSTOM_INSTRUCTIONS": true,
		}
		if configKeys[req.Key] {
			s.Engine.UpdateConfig(req.Key, req.Value)
			saveEnv(req.Key, req.Value)
		}

		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		key := r.URL.Query().Get("key")
		s.Engine.DB.DeleteSecret(key)
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handleSkills(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		name := r.URL.Query().Get("name")
		if name != "" {
			docs, err := s.Engine.DB.GetSkill(name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(docs))
			return
		}

		skills, _ := s.Engine.DB.ListSkills()
		if skills == nil {
			skills = []string{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(skills)

	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
			Docs string `json:"docs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		err := s.Engine.DB.SaveSkill(req.Name, req.Docs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.Engine.RefreshSystemPrompt()
		w.WriteHeader(http.StatusCreated)

	case http.MethodDelete:
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Missing name", http.StatusBadRequest)
			return
		}
		err := s.Engine.DB.DeleteSkill(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.Engine.RefreshSystemPrompt()
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	
	// Remove leading slash for embed.FS
	embedPath := "ui" + path
	data, err := uiFS.ReadFile(embedPath)
	if err != nil {
		// Fallback to index.html for unknown paths (SPA behavior)
		data, err = uiFS.ReadFile("ui/index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}

	if strings.HasSuffix(path, ".png") {
		w.Header().Set("Content-Type", "image/png")
	} else if strings.HasSuffix(path, ".html") {
		w.Header().Set("Content-Type", "text/html")
	} else if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript")
	} else if strings.HasSuffix(path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}

	w.Write(data)
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
