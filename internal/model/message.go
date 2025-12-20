package model

import (
	"time"
)

type Message struct {
	ID        string `json:"id"`
	RoomID    string `json:"roomId"`
	UserID    string `json:"userId"`
	Content   string `json:"content"`
	Type      string `json:"type"` // text, image, system
	Timestamp int64  `json:"timestamp"`
	Avatar    string `json:"avatar"`
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	RoomID   string `json:"roomId"`
}

type Room struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UserCount int       `json:"userCount"`
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
