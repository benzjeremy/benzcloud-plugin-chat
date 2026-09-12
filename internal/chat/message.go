package chat

import "time"

// MessageType indicates the nature of the message.
type MessageType string

const (
	TypeMessage MessageType = "message"
	TypeJoin    MessageType = "join"
	TypeLeave   MessageType = "leave"
	TypeSystem  MessageType = "system"
	TypePing    MessageType = "ping"
	TypePong    MessageType = "pong"
)

// Message models a chat event or payload.
type Message struct {
	ID        string      `json:"id"`
	Channel   string      `json:"channel"`
	Sender    string      `json:"sender"`
	Content   string      `json:"content"`
	Type      MessageType `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
}

// Channel models a chat room.
type Channel struct {
	Name        string `json:"name"`
	Topic       string `json:"topic"`
	IsDirect    bool   `json:"is_direct"`
	CreatedBy   string `json:"created_by"`
}
