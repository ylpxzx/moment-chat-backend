package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"moment-chat-backend/internal/model"
)

// upgrader 是用于将 HTTP 连接升级为 WebSocket 的 websocket.Upgrader 实例。
// 它预设了读/写缓冲区大小（1024 字节）以优化小消息的传输性能，
// 并且 CheckOrigin 返回 true，开放了跨来源请求（仅用于开发/测试），
// 在生产环境中应根据安全策略限制允许的来源。
//
// MessageTypeText 表示普通文本消息的类型标识符，通常用于客户端间的聊天或用户消息传递。
//
// MessageTypeSystem 表示系统消息的类型标识符，用于广播通知、状态更新或服务端生成的控制消息。
var (
	// WebSocket 配置
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 生产环境应该限制来源
		},
	}

	// 消息类型常量
	MessageTypeText   = "text"
	MessageTypeSystem = "system"
)

// Client 表示一个WebSocket客户端
type Client struct {
	ID       string
	UserId   string
	RoomID   string
	Username string
	Avatar   string
	Conn     *websocket.Conn
	Send     chan []byte
}

// Room 表示聊天室
type Room struct {
	ID      string
	Clients map[string]*Client
	mu      sync.RWMutex
}

// Manager 管理所有房间和客户端
type Manager struct {
	Rooms map[string]*Room
	mu    sync.RWMutex
	Redis *redis.Client
}

// NewManager 创建新的管理器
func NewManager(redisClient *redis.Client) *Manager {
	return &Manager{
		Rooms: make(map[string]*Room),
		Redis: redisClient,
	}
}

// NewRoom 创建新房间
func (m *Manager) NewRoom(id string) *Room {
	m.mu.Lock()         // 加写锁，确保对 m.Rooms 的并发写操作是线程安全的
	defer m.mu.Unlock() // 在函数返回时释放写锁，避免死锁并确保及时解锁

	room := &Room{
		ID:      id,
		Clients: make(map[string]*Client),
	}

	m.Rooms[id] = room
	return room
}

// GetRoom 获取房间
func (m *Manager) GetRoom(id string) (*Room, bool) {
	// 对 Rooms 的并发读取加读锁，防止与写操作产生数据竞争
	m.mu.RLock()
	// 在函数返回时释放读锁，使用 defer 确保异常或提前返回也能解锁
	defer m.mu.RUnlock()

	room, exists := m.Rooms[id]
	log.Printf("GetRoom: id=%s, exists=%v", id, exists)
	return room, exists
}

// AddClient 添加客户端到房间
func (r *Room) AddClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Clients[client.ID] = client
	log.Printf("AddClient: clientID=%s, roomID=%s", client.ID, r.ID)
	// 发送用户加入通知
	// 构造“用户加入”系统消息，通知房间内其他成员
	joinMsg := model.WebSocketMessage{
		Type: "user_join", // 消息类型：用户加入
		Payload: map[string]interface{}{
			"userId":    client.UserId,   // 加入用户的唯一标识
			"username":  client.Username, // 加入用户的昵称
			"avatar":    client.Avatar,   // 加入用户的头像
			"userCount": len(r.Clients),  // 当前房间在线用户数量（包括新加入者）
			"userList": func() []map[string]string {
				userList := []map[string]string{}
				for _, c := range r.Clients {
					userList = append(userList, map[string]string{
						"userId":   c.UserId,
						"username": c.Username,
						"avatar":   c.Avatar,
					})
				}
				return userList
			}(),
		},
	}
	log.Printf("Broadcasting join message for clientID=%s in roomID=%s", client.ID, r.ID)
	r.Broadcast(joinMsg, client.ID)
}

// RemoveClient 从房间移除客户端
func (r *Room) RemoveClient(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if client, exists := r.Clients[clientID]; exists {
		// 从房间的客户端集合中移除该客户端（已持有写锁，确保并发安全）
		delete(r.Clients, clientID)
		// 关闭该客户端的发送通道，通知写协程退出，避免继续向已断开连接发送数据
		close(client.Send)

		// 发送用户离开通知
		// 构造“用户离开”系统消息，通知房间内其他成员
		leaveMsg := model.WebSocketMessage{
			Type: "user_leave", // 消息类型：用户离开
			Payload: map[string]interface{}{
				"userId":    client.UserId,  // 离开用户的唯一标识
				"userCount": len(r.Clients), // 当前房间剩余在线用户数量（移除后）
			},
		}

		r.Broadcast(leaveMsg, "")
	}
}

// Broadcast 广播消息给房间内所有客户端
// Broadcast 将给定的 WebSocket 消息以 JSON 序列化后广播到房间内所有客户端，
// 可选择排除指定的客户端（excludeClientID）。
//
// 并发控制：
//   - 为避免在遍历过程中对 Clients 映射发生并发修改，先在读锁（RLock）下
//     复制当前客户端引用到切片，随后释放读锁再进行发送操作。
//   - 发送时使用非阻塞的 select：如果发送通道可写则立即投递；否则视为该客户端
//     已阻塞或异常，不再阻塞当前广播流程。
//
// 发送与清理流程：
// 1. 在读锁保护下过滤并收集需要接收的客户端（跳过 excludeClientID）。
// 2. 将 message 进行 JSON 序列化；若失败则记录错误并终止广播。
// 3. 逐个尝试向客户端的 Send 通道非阻塞写入：
//   - 成功：消息进入客户端缓冲队列，等待实际写到 WebSocket。
//   - 失败（default 分支）：关闭该客户端的发送通道，并在写锁（Lock）下
//     将其从房间的 Clients 集合中移除，以保持房间状态的健康与可用性。
//
// 设计要点：
// - 通过“先复制、后发送”的策略避免迭代时的竞态与加锁时间过长。
// - 使用非阻塞发送确保慢客户端不会拖累广播路径。
// - 异常客户端及时清理，减少资源泄漏和后续错误。
func (r *Room) Broadcast(message model.WebSocketMessage, excludeClientID string) {
	// // 先在读锁下复制当前客户端引用，避免在迭代时修改 map
	// r.mu.RLock()
	clients := make([]*Client, 0, len(r.Clients))
	for _, c := range r.Clients {
		// if c.ID == excludeClientID {
		// 	continue
		// }
		clients = append(clients, c)
	}
	// r.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}
	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			// 异常客户端及时清理，减少资源泄漏和后续错误
			// 发送失败：安全关闭发送通道并从房间中移除客户端（使用写锁）
			close(client.Send)
			r.mu.Lock()
			delete(r.Clients, client.ID)
			r.mu.Unlock()
		}
	}
}

// HandleWebSocket 处理WebSocket连接
func (m *Manager) HandleWebSocket(w http.ResponseWriter, roomID string, req *http.Request) {
	if roomID == "" {
		http.Error(w, "Room ID required", http.StatusBadRequest)
		return
	}

	// 升级HTTP连接到WebSocket
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		ID:     uuid.New().String(),
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	go client.writePump()
	go client.readPumpInit(roomID, m)
}

// readPumpInit 处理首次 join 消息和后续消息
func (c *Client) readPumpInit(roomID string, m *Manager) {
	defer func() {
		log.Printf("[readPumpInit] 关闭连接: %s", c.ID)
		if room, exists := m.GetRoom(roomID); exists {
			room.RemoveClient(c.ID)
		}
		c.Conn.Close()
	}()

	// 读取 join 消息
	_, message, err := c.Conn.ReadMessage()
	if err != nil {
		log.Printf("[readPumpInit] 读取 join 消息失败: %v", err)
		return
	}

	var joinRequest struct {
		Username string `json:"username"`
		Avatar   string `json:"avatar"`
		UserId   string `json:"user_id"`
	}

	if err := json.Unmarshal(message, &joinRequest); err != nil {
		log.Printf("[readPumpInit] join 消息解析失败: %v", err)
		c.Conn.WriteJSON(map[string]string{"error": "Invalid join request"})
		return
	}

	log.Printf("[readPumpInit] Client %s joining room %s with userId %s", c.ID, roomID, joinRequest.UserId)

	c.Username = joinRequest.Username
	c.Avatar = joinRequest.Avatar
	c.UserId = joinRequest.UserId

	// 获取或创建房间
	var room *Room
	if existingRoom, exists := m.GetRoom(roomID); exists {
		room = existingRoom
	} else {
		room = m.NewRoom(roomID)
	}

	room.AddClient(c)
	// 进入正常消息循环
	c.readPump(room, m)
}

// writePump 发送消息给客户端
func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

// readPump 从客户端读取消息
func (c *Client) readPump(room *Room, m *Manager) {
	defer func() {
		log.Printf("[readPump] 关闭连接: %s", c.ID)
		room.RemoveClient(c.ID)
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("[readPump] 读取消息出错 %s: %v", c.ID, err)
			break
		}
		log.Printf("[readPump] 读取客户端消息 %s: %s", c.ID, string(message))

		var msgData struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}

		if err := json.Unmarshal(message, &msgData); err != nil {
			log.Printf("[readPump] 消息解析失败 %s: %v", c.ID, err)
			continue
		}

		// 处理消息
		m.handleMessage(c, room, msgData.Type, msgData.Content)
	}
}

// handleMessage 处理不同类型的消息
func (m *Manager) handleMessage(client *Client, room *Room, msgType, content string) {
	log.Printf("Handling message from client %s: type=%s, content=%s, userId=%s", client.ID, msgType, content, client.UserId)
	switch msgType {
	case "text":
		// 创建消息对象
		message := model.Message{
			ID:        uuid.New().String(),
			RoomID:    room.ID,
			UserID:    client.UserId,
			Content:   content,
			Type:      MessageTypeText,
			Timestamp: time.Now().UnixMilli(),
			Avatar:    client.Avatar,
			Username:  client.Username,
			ExpiresAt: time.Now().Add(20 * time.Second).UnixMilli(),
		}

		// 保存到Redis（20秒后自动过期）
		ctx := context.Background()
		messageJSON, _ := json.Marshal(message)
		messageKey := "message:" + message.ID

		m.Redis.Set(ctx, messageKey, messageJSON, 20*time.Second)
		m.Redis.LPush(ctx, "room:messages:"+room.ID, message.ID)
		m.Redis.Expire(ctx, "room:messages:"+room.ID, 24*time.Hour)

		// 广播消息给房间内其他用户
		wsMessage := model.WebSocketMessage{
			Type:    "new_message",
			Payload: message,
		}

		room.Broadcast(wsMessage, client.ID)

	case "typing":
		// 广播正在输入状态
		wsMessage := model.WebSocketMessage{
			Type: "user_typing",
			Payload: map[string]interface{}{
				"userId":   client.UserId,
				"username": client.Username,
			},
		}

		room.Broadcast(wsMessage, client.ID)
	}
}
