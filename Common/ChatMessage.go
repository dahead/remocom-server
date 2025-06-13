package common

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

type MessageType string

const (
	TypeAuth  MessageType = "auth"
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

func NewAuthenticateMessage(username, accessCode string) *ChatMessage {
	return &ChatMessage{
		MessageID: generateUniqueID(),
		Timestamp: time.Now(),
		Username:  username,
		Content:   accessCode,
		Type:      TypeAuth,
	}
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

func (m *ChatMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

func FromJSON(data []byte) (*ChatMessage, error) {
	var msg ChatMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func GenerateKeyFromAccessCode(accessCode string) []byte {
	hash := sha256.Sum256([]byte(accessCode))
	return hash[:]
}

func (m *ChatMessage) ToEncryptedJSON(accessCode string) ([]byte, error) {
	jsonData, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	key := GenerateKeyFromAccessCode(accessCode)
	return Encrypt(key, jsonData)
}

func FromEncryptedJSON(data []byte, accessCode string) (*ChatMessage, error) {
	key := GenerateKeyFromAccessCode(accessCode)
	decryptedData, err := Decrypt(key, data)
	if err != nil {
		return nil, err
	}

	var msg ChatMessage
	err = json.Unmarshal(decryptedData, &msg)
	return &msg, err
}
