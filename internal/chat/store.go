package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Store handles persistence for chat channels and message logs.
type Store struct {
	dataDir  string
	channels map[string]*Channel
	history  map[string][]*Message // keyed by channel name
	mu       sync.RWMutex
}

// NewStore initializes channel and message storage.
func NewStore(dataDir string) (*Store, error) {
	chDir := filepath.Join(dataDir, "chat_history")
	if err := os.MkdirAll(chDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create chat directory: %w", err)
	}

	s := &Store{
		dataDir:  chDir,
		channels: make(map[string]*Channel),
		history:  make(map[string][]*Message),
	}

	// Setup default channels
	s.channels["general"] = &Channel{Name: "general", Topic: "General team discussion and announcements", CreatedBy: "system"}
	s.channels["dev"] = &Channel{Name: "dev", Topic: "Engineering, git commits, and tech topics", CreatedBy: "system"}
	s.channels["random"] = &Channel{Name: "random", Topic: "Casual coffee break and memes", CreatedBy: "system"}

	_ = s.loadAll()
	return s, nil
}

func (s *Store) channelFile(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.ReplaceAll(clean, "/", "_")
	return filepath.Join(s.dataDir, clean+".json")
}

func (s *Store) loadAll() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			chName := strings.TrimSuffix(e.Name(), ".json")
			data, err := os.ReadFile(filepath.Join(s.dataDir, e.Name()))
			if err == nil {
				var msgs []*Message
				if err := json.Unmarshal(data, &msgs); err == nil {
					s.history[chName] = msgs
					if _, exists := s.channels[chName]; !exists {
						s.channels[chName] = &Channel{Name: chName, Topic: "Discussions in #" + chName, CreatedBy: "user"}
					}
				}
			}
		}
	}
	return nil
}

func (s *Store) saveChannel(name string) error {
	msgs := s.history[name]
	data, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return err
	}
	cFile := s.channelFile(name)
	tmp := cFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, cFile)
}

// SaveMessage appends a message to the channel history.
func (s *Store) SaveMessage(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := strings.ToLower(msg.Channel)
	if msg.ID == "" {
		msg.ID = fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}

	s.history[ch] = append(s.history[ch], msg)
	if _, exists := s.channels[ch]; !exists {
		s.channels[ch] = &Channel{Name: ch, Topic: "Discussions in #" + ch, CreatedBy: msg.Sender}
	}

	return s.saveChannel(ch)
}

// GetMessages retrieves recent messages for a channel.
func (s *Store) GetMessages(channel string, limit int) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := strings.ToLower(channel)
	msgs := s.history[ch]
	if len(msgs) == 0 {
		return []*Message{}
	}

	if limit <= 0 || limit > len(msgs) {
		limit = len(msgs)
	}

	start := len(msgs) - limit
	result := make([]*Message, limit)
	for i := 0; i < limit; i++ {
		copyM := *msgs[start+i]
		result[i] = &copyM
	}
	return result
}

// GetChannels returns list of all channels.
func (s *Store) GetChannels() []*Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*Channel
	for _, c := range s.channels {
		copyC := *c
		list = append(list, &copyC)
	}
	return list
}

// CreateChannel creates a new channel if it doesn't already exist.
func (s *Store) CreateChannel(name, topic, createdBy string) (*Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.TrimPrefix(clean, "#")
	if clean == "" {
		return nil, fmt.Errorf("channel name cannot be empty")
	}

	if ch, exists := s.channels[clean]; exists {
		return ch, nil
	}

	ch := &Channel{
		Name:      clean,
		Topic:     topic,
		CreatedBy: createdBy,
	}
	s.channels[clean] = ch
	_ = s.saveChannel(clean)
	return ch, nil
}
