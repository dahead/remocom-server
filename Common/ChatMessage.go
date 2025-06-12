package common

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

type MessageType string

const (
	TypeChat  MessageType = "chat"
	TypePing  MessageType = "ping"
	TypeAlive MessageType = "alive"
)

type ChatMessage struct {
	MessageID string      `json:"message_id"`
	Username  string      `json:"username"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Type      MessageType `json:"type"`
}

func NewChatMessage(username, content string) *ChatMessage {
	return &ChatMessage{
		MessageID: generateUniqueID(),
		Timestamp: time.Now(),
		Username:  username,
		Content:   content,
		Type:      TypeChat,
	}
}

// Add helper functions for ping/alive messages
func NewPingMessage() *ChatMessage {
	return &ChatMessage{
		MessageID: generateUniqueID(),
		Timestamp: time.Now(),
		Type:      TypePing,
	}
}

func NewAliveMessage(username string) *ChatMessage {
	return &ChatMessage{
		MessageID: generateUniqueID(),
		Timestamp: time.Now(),
		Username:  username,
		Type:      TypeAlive,
	}
}

func generateUniqueID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return time.Now().Format("20060102150405") + hex.EncodeToString(b)
}

// ToJSON serializes the ChatMessage struct to JSON bytes
func (m *ChatMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON deserializes JSON bytes to a ChatMessage struct
func FromJSON(data []byte) (*ChatMessage, error) {
	var msg ChatMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}
