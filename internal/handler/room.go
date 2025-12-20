package handler

import (
	"math/rand"
	"moment-chat-backend/pkg/websocket"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	wsManager *websocket.Manager
}

func NewRoomHandler(wsManager *websocket.Manager) *RoomHandler {
	return &RoomHandler{
		wsManager: wsManager,
	}
}

// CreateRoom 创建新房间
func (h *RoomHandler) CreateRoom(c *gin.Context) {
	roomID := generateRoomID(6)

	room := h.wsManager.NewRoom(roomID)
	if room == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create room",
		})
		return
	}

	response := map[string]interface{}{
		"roomId":    roomID,
		"createdAt": time.Now().Unix(),
		"message":   "Room created successfully",
	}

	c.JSON(http.StatusOK, response)
}

// CheckRoom 检查房间是否存在
func (h *RoomHandler) CheckRoom(c *gin.Context) {
	roomID := c.Param("roomId")

	// 检查房间号是否合法（只允许字母和数字，长度为6）
	if len(roomID) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}
	for _, ch := range roomID {
		if !(ch >= 'a' && ch <= 'z') && !(ch >= 'A' && ch <= 'Z') && !(ch >= '0' && ch <= '9') {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid room ID",
			})
			return
		}
	}

	if _, exists := h.wsManager.GetRoom(roomID); exists {
		c.JSON(http.StatusOK, gin.H{
			"exists": true,
			"roomId": roomID,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
			"roomId": roomID,
		})
	}
}

// GetRoomInfo 获取房间信息
func (h *RoomHandler) GetRoomInfo(c *gin.Context) {
	roomID := c.Param("roomId")

	if room, exists := h.wsManager.GetRoom(roomID); exists {
		c.JSON(http.StatusOK, gin.H{
			"roomId":    room.ID,
			"userCount": len(room.Clients),
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Room not found",
		})
	}
}

// GetDefaultAvatars 获取默认头像列表
func (h *RoomHandler) GetDefaultAvatars(c *gin.Context) {
	avatars := []string{
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Jocelyn",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Liam",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Sawyer",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Katherine",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Easton",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Christian",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Eliza",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Aiden",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Vivian",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Alexander",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Caleb",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Jack",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Brooklynn",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Andrea",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Maria",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Aidan",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Riley",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Jessica",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Sarah",
		"https://api.dicebear.com/9.x/bottts-neutral/svg?seed=Mackenzie",
	}

	c.JSON(http.StatusOK, gin.H{
		"avatars": avatars,
	})
}

// generateRoomID 生成随机房间ID
func generateRoomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
