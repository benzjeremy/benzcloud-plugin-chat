package chat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Local mesh network trusted origin
	},
}

// Client represents a connected user session.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
}

// Hub coordinates all active client connections and broadcasts.
type Hub struct {
	store      *Store
	clients    map[*Client]bool
	userCounts map[string]int
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new chat hub.
func NewHub(store *Store) *Hub {
	return &Hub{
		store:      store,
		clients:    make(map[*Client]bool),
		userCounts: make(map[string]int),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run executes the main event coordination loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.userCounts[client.username]++
			h.mu.Unlock()

			// Broadcast presence update
			h.broadcastPresence()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.userCounts[client.username]--
				if h.userCounts[client.username] <= 0 {
					delete(h.userCounts, client.username)
				}
			}
			h.mu.Unlock()

			// Broadcast presence update
			h.broadcastPresence()

		case msg := <-h.broadcast:
			// Save message
			_ = h.store.SaveMessage(msg)

			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) broadcastPresence() {
	h.mu.RLock()
	users := make([]string, 0, len(h.userCounts))
	for u := range h.userCounts {
		users = append(users, u)
	}
	h.mu.RUnlock()

	presenceMsg := map[string]interface{}{
		"type":  "presence",
		"users": users,
	}
	data, _ := json.Marshal(presenceMsg)

	h.mu.RLock()
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
		}
	}
	h.mu.RUnlock()
}

// GetOnlineUsers returns list of currently connected user handles.
func (h *Hub) GetOnlineUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]string, 0, len(h.userCounts))
	for u := range h.userCounts {
		users = append(users, u)
	}
	return users
}

// Broadcast sends a message to the hub loop.
func (h *Hub) Broadcast(msg *Message) {
	h.broadcast <- msg
}

// ServeWS handles incoming WebSocket upgrade requests.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, username string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[BenzCloud Chat] WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(65536)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg.Sender == "" {
				msg.Sender = c.username
			}
			if msg.Timestamp.IsZero() {
				msg.Timestamp = time.Now().UTC()
			}
			if msg.Type == "" {
				msg.Type = TypeMessage
			}
			c.hub.broadcast <- &msg
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
