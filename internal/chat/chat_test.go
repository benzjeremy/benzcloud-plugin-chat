package chat

import (
	"os"
	"testing"
	"time"
)

func TestChatStore(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chat_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Verify default channels
	channels := store.GetChannels()
	if len(channels) < 3 {
		t.Fatalf("expected at least 3 channels, got %d", len(channels))
	}

	// Create custom channel
	ch, err := store.CreateChannel("marketing", "Marketing strategies", "alice")
	if err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}
	if ch.Name != "marketing" {
		t.Fatalf("expected channel name 'marketing', got %s", ch.Name)
	}

	// Save message
	msg := &Message{
		Channel:   "marketing",
		Sender:    "alice",
		Content:   "Launch campaign is ready!",
		Type:      TypeMessage,
		Timestamp: time.Now().UTC(),
	}
	if err := store.SaveMessage(msg); err != nil {
		t.Fatalf("failed to save message: %v", err)
	}

	// Retrieve messages
	msgs := store.GetMessages("marketing", 10)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Launch campaign is ready!" {
		t.Fatalf("unexpected content: %s", msgs[0].Content)
	}

	// Test persistence reload
	store2, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to reload store: %v", err)
	}
	msgs2 := store2.GetMessages("marketing", 10)
	if len(msgs2) != 1 {
		t.Fatalf("expected 1 reloaded message, got %d", len(msgs2))
	}
}

func TestHubOnlineUsers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hub_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, _ := NewStore(tmpDir)
	hub := NewHub(store)
	go hub.Run()

	users := hub.GetOnlineUsers()
	if len(users) != 0 {
		t.Fatalf("expected 0 online users, got %d", len(users))
	}
}
