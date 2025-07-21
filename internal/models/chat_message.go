package models

import "time"

// ChatMessage represents a message in a chat conversation.
type ChatMessage struct {
	ID             int64 // Primary key
	UserID         int64 // User.ID of the client
	OperatorID     int64 // User.ID of the operator
	Message        string
	IsFromUser     bool
	ConversationID string    // To group messages in a conversation
	CreatedAt      time.Time // from DB schema
}
