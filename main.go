package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/benzjeremy/benzcloud-plugin-chat/internal/chat"
)

//go:embed web/*
var webFS embed.FS

func main() {
	port := flag.Int("port", 8093, "HTTP & WebSocket server port")
	dataDir := flag.String("data", "./data", "Directory for chat history")
	baseDomain := flag.String("domain", "benzcloud.local", "Base domain for chat")
	flag.Parse()

	log.Printf("[BenzCloud Chat] Initializing Chat Plugin v1.0 on %s (Port %d)\n", *baseDomain, *port)

	store, err := chat.NewStore(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize chat store: %v", err)
	}

	// Seed welcome message if #general is empty
	genMsgs := store.GetMessages("general", 1)
	if len(genMsgs) == 0 {
		_ = store.SaveMessage(&chat.Message{
			Channel:   "general",
			Sender:    "BenzCloud Bot",
			Content:   "Welcome to your private, decentralized Mesh Chat! Real-time messaging with end-to-end local sovereignty.",
			Type:      chat.TypeMessage,
			Timestamp: time.Now().UTC(),
		})
	}

	hub := chat.NewHub(store)
	go hub.Run()

	mux := http.NewServeMux()

	// Health endpoint for BenzCloud Supervisor / Router
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"plugin":    "chat",
			"version":   "v1.0",
			"subdomain": "chat",
			"port":      *port,
		})
	})

	// WebSocket handler
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		if user == "" {
			user = "anonymous"
		}
		hub.ServeWS(w, r, user)
	})

	// API: Channels
	mux.HandleFunc("/api/chat/channels", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			channels := store.GetChannels()
			_ = json.NewEncoder(w).Encode(channels)
			return
		}

		if r.Method == http.MethodPost {
			var body struct {
				Name      string `json:"name"`
				Topic     string `json:"topic"`
				CreatedBy string `json:"created_by"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			ch, err := store.CreateChannel(body.Name, body.Topic, body.CreatedBy)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(ch)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// API: Messages
	mux.HandleFunc("/api/chat/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			ch := r.URL.Query().Get("channel")
			if ch == "" {
				ch = "general"
			}
			limit := 50
			if lStr := r.URL.Query().Get("limit"); lStr != "" {
				if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
					limit = l
				}
			}
			msgs := store.GetMessages(ch, limit)
			_ = json.NewEncoder(w).Encode(msgs)
			return
		}

		if r.Method == http.MethodPost {
			var msg chat.Message
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if msg.Channel == "" {
				msg.Channel = "general"
			}
			if msg.Sender == "" {
				msg.Sender = "anonymous"
			}
			if msg.Timestamp.IsZero() {
				msg.Timestamp = time.Now().UTC()
			}
			hub.Broadcast(&msg)
			_ = json.NewEncoder(w).Encode(msg)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	// Static Web Chat Files
	subWeb, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(subWeb))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/ws" && r.URL.Path != "/health" {
			fileServer.ServeHTTP(w, r)
			return
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: mux,
	}

	go func() {
		log.Printf("[BenzCloud Chat] Web Chat & WebSocket listening on http://0.0.0.0:%d\n", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Chat HTTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[BenzCloud Chat] Shutting down gracefully...")
	_ = server.Close()
}
