package unit

import (
	"testing"
	"time"

	"moment-chat-backend/internal/model"

	"github.com/stretchr/testify/assert"
)

func TestMessageModel(t *testing.T) {
	now := time.Now()
	msg := model.Message{
		ID:        "msg123",
		RoomID:    "room456",
		UserID:    "user789",
		Content:   "Hello, World! 😊",
		Type:      "text",
		Timestamp: now.UnixMilli(),
		Avatar:    "https://example.com/avatar.png",
		Username:  "TestUser",
		ExpiresAt: now.Add(20 * time.Second).UnixMilli(),
	}

	assert.Equal(t, "msg123", msg.ID)
	assert.Equal(t, "room456", msg.RoomID)
	assert.Equal(t, "user789", msg.UserID)
	assert.Equal(t, "Hello, World! 😊", msg.Content)
	assert.Equal(t, "text", msg.Type)
	assert.Equal(t, now.UnixMilli(), msg.Timestamp)
	assert.Equal(t, "https://example.com/avatar.png", msg.Avatar)
	assert.Equal(t, "TestUser", msg.Username)
	assert.Equal(t, now.Add(20*time.Second).UnixMilli(), msg.ExpiresAt)
}

func TestUserModel(t *testing.T) {
	user := model.User{
		ID:       "user123",
		Username: "TestUser",
		Avatar:   "avatar.png",
		RoomID:   "room456",
	}

	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "TestUser", user.Username)
	assert.Equal(t, "avatar.png", user.Avatar)
	assert.Equal(t, "room456", user.RoomID)
}

func TestRoomModel(t *testing.T) {
	now := time.Now()
	room := model.Room{
		ID:        "room123",
		CreatedAt: now,
		UserCount: 5,
	}

	assert.Equal(t, "room123", room.ID)
	assert.Equal(t, now.Unix(), room.CreatedAt.Unix())
	assert.Equal(t, 5, room.UserCount)
}

func TestWebSocketMessage(t *testing.T) {
	// 测试文本消息
	wsMsg := model.WebSocketMessage{
		Type: "new_message",
		Payload: model.Message{
			ID:      "msg1",
			Content: "Hello",
		},
	}

	assert.Equal(t, "new_message", wsMsg.Type)

	// 测试系统消息
	wsMsg2 := model.WebSocketMessage{
		Type: "user_join",
		Payload: map[string]interface{}{
			"userId":   "user1",
			"username": "TestUser",
		},
	}

	assert.Equal(t, "user_join", wsMsg2.Type)
}
