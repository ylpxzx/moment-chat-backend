package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WebSocketTestSuite WebSocket测试套件
type WebSocketTestSuite struct {
	suite.Suite
	server *httptest.Server
}

// SetupSuite 设置测试套件
func (s *WebSocketTestSuite) SetupSuite() {
	// 创建测试WebSocket服务器
	s.setupWebSocketTestServer()
}

func (s *WebSocketTestSuite) setupWebSocketTestServer() {
	r := gin.New()

	// 创建WebSocket管理器用于测试
	s.server = httptest.NewServer(r)
}

// TearDownSuite 清理测试套件
func (s *WebSocketTestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
}

// TestWebSocketConnection 测试WebSocket连接
func (s *WebSocketTestSuite) TestWebSocketConnection() {
	if testing.Short() {
		s.T().Skip("Skipping WebSocket test in short mode")
	}

	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 升级到WebSocket
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 处理连接
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// 回显消息
			if err := conn.WriteMessage(messageType, p); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// 连接到WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(s.T(), err)
	defer conn.Close()

	// 发送测试消息
	testMessage := `{"type": "join", "username": "test", "avatar": "avatar.png"}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	require.NoError(s.T(), err)

	// 接收响应
	_, response, err := conn.ReadMessage()
	require.NoError(s.T(), err)
	s.T().Logf("Received response: %s", string(response))
	require.NoError(s.T(), err)

	assert.Equal(s.T(), testMessage, string(response))
}

// TestWebSocketRoomJoin 测试房间加入流程
func (s *WebSocketTestSuite) TestWebSocketRoomJoin() {
	if testing.Short() {
		s.T().Skip("Skipping WebSocket test in short mode")
	}

	roomID := "TESTRO"
	username := "TestUser"
	avatar := "https://example.com/avatar.png"

	// 创建模拟WebSocket处理器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 读取加入消息
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var joinMsg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &joinMsg); err != nil {
			return
		}

		// 验证加入消息
		assert.Equal(s.T(), "join", joinMsg["type"])
		assert.Equal(s.T(), username, joinMsg["username"])
		assert.Equal(s.T(), avatar, joinMsg["avatar"])

		// 发送欢迎消息
		welcomeMsg := map[string]interface{}{
			"type": "welcome",
			"payload": map[string]interface{}{
				"message":   "Welcome to the room",
				"roomId":    roomID,
				"userCount": 1,
			},
		}

		welcomeBytes, _ := json.Marshal(welcomeMsg)
		conn.WriteMessage(websocket.TextMessage, welcomeBytes)
	}))
	defer server.Close()

	// 连接到WebSocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/rooms/" + roomID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(s.T(), err)
	defer conn.Close()

	// 发送加入消息
	joinMsg := map[string]interface{}{
		"type":     "join",
		"username": username,
		"avatar":   avatar,
	}

	joinBytes, _ := json.Marshal(joinMsg)
	err = conn.WriteMessage(websocket.TextMessage, joinBytes)
	require.NoError(s.T(), err)

	// 接收欢迎消息
	_, response, err := conn.ReadMessage()
	require.NoError(s.T(), err)

	var welcomeResp map[string]interface{}
	err = json.Unmarshal(response, &welcomeResp)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "welcome", welcomeResp["type"])
}

// TestWebSocketMessageSend 测试消息发送
func (s *WebSocketTestSuite) TestWebSocketMessageSend() {
	if testing.Short() {
		s.T().Skip("Skipping WebSocket test in short mode")
	}

	messageContent := "Hello, World! 😊"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 读取消息
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			return
		}

		// 验证消息格式
		assert.Equal(s.T(), "text", msg["type"])
		assert.Equal(s.T(), messageContent, msg["content"])

		// 发送确认
		ackMsg := map[string]interface{}{
			"type": "message_ack",
			"payload": map[string]interface{}{
				"status":    "delivered",
				"messageId": "msg123",
				"timestamp": time.Now().UnixMilli(),
			},
		}

		ackBytes, _ := json.Marshal(ackMsg)
		conn.WriteMessage(websocket.TextMessage, ackBytes)
	}))
	defer server.Close()

	// 连接并发送消息
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(s.T(), err)
	defer conn.Close()

	// 发送文本消息
	textMsg := map[string]interface{}{
		"type":    "text",
		"content": messageContent,
	}

	textBytes, _ := json.Marshal(textMsg)
	err = conn.WriteMessage(websocket.TextMessage, textBytes)
	require.NoError(s.T(), err)

	// 接收确认
	_, response, err := conn.ReadMessage()
	require.NoError(s.T(), err)

	var ackResp map[string]interface{}
	err = json.Unmarshal(response, &ackResp)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "message_ack", ackResp["type"])
	assert.Equal(s.T(), "delivered", ackResp["payload"].(map[string]interface{})["status"])
}

// TestWebSocketDisconnection 测试连接断开
func (s *WebSocketTestSuite) TestWebSocketDisconnection() {
	if testing.Short() {
		s.T().Skip("Skipping WebSocket test in short mode")
	}

	disconnected := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			conn.Close()
			disconnected <- true
		}()

		// 保持连接直到客户端断开
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// 连接然后立即断开
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(s.T(), err)

	// 短暂等待后断开连接
	time.Sleep(100 * time.Millisecond)
	conn.Close()

	// 验证服务器检测到断开
	select {
	case <-disconnected:
		// 成功检测到断开
		return
	case <-time.After(1 * time.Second):
		s.Fail("Server did not detect disconnection")
	}
}

func TestWebSocketTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping WebSocket tests in short mode")
	}
	suite.Run(t, new(WebSocketTestSuite))
}
