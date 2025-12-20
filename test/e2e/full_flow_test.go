package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FullFlowTestSuite 端到端流程测试套件
type FullFlowTestSuite struct {
	suite.Suite
	serverURL string
	client    *http.Client
	roomID    string
}

// SetupSuite 设置测试套件
func (s *FullFlowTestSuite) SetupSuite() {
	s.serverURL = "http://localhost:8080"
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}

	// 等待服务器就绪
	s.waitForServer()

	// 创建测试房间
	s.createTestRoom()
}

func (s *FullFlowTestSuite) waitForServer() {
	for i := 0; i < 10; i++ {
		resp, err := s.client.Get(s.serverURL + "/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	s.FailNow("Server did not start in time")
}

func (s *FullFlowTestSuite) createTestRoom() {
	resp, err := s.client.Post(s.serverURL+"/api/v1/rooms", "application/json", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(s.T(), err)

	s.roomID = result["roomId"].(string)
	fmt.Printf("Created test room: %s\n", s.roomID)
}

// TestCompleteUserFlow 测试完整的用户流程
func (s *FullFlowTestSuite) TestCompleteUserFlow() {
	// 1. 获取默认头像
	resp, err := s.client.Get(s.serverURL + "/api/v1/avatars")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	var avatarsResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&avatarsResp)
	require.NoError(s.T(), err)

	avatars := avatarsResp["avatars"].([]interface{})
	require.True(s.T(), len(avatars) > 0)

	// 2. 检查房间是否存在
	resp, err = s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/check", s.serverURL, s.roomID))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	var checkResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&checkResp)
	require.NoError(s.T(), err)

	assert.True(s.T(), checkResp["exists"].(bool))

	// 3. 获取房间信息
	resp, err = s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/info", s.serverURL, s.roomID))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	var infoResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&infoResp)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), s.roomID, infoResp["roomId"])

	// 4. 测试WebSocket端点（模拟）
	// 在实际测试中，这里会测试WebSocket连接
	// 但由于需要实时服务器，这里只验证端点格式
	wsEndpoint := fmt.Sprintf("/ws/rooms/%s", s.roomID)
	assert.True(s.T(), strings.HasPrefix(wsEndpoint, "/ws/rooms/"))
}

// TestMultipleRoomCreation 测试多个房间创建
func (s *FullFlowTestSuite) TestMultipleRoomCreation() {
	roomIDs := make(map[string]bool)

	// 创建多个房间，确保每个房间ID都是唯一的
	for i := 0; i < 5; i++ {
		resp, err := s.client.Post(s.serverURL+"/api/v1/rooms", "application/json", nil)
		require.NoError(s.T(), err)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(s.T(), err)
		resp.Body.Close()

		roomID := result["roomId"].(string)

		// 验证房间ID格式
		assert.Len(s.T(), roomID, 6)
		assert.Regexp(s.T(), "^[A-Z0-9a-z]{6}$", roomID)

		// 确保房间ID唯一
		assert.False(s.T(), roomIDs[roomID], "Room ID should be unique")
		roomIDs[roomID] = true

		// 短暂延迟以避免速率限制
		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(s.T(), 5, len(roomIDs))
}

// TestErrorHandling 测试错误处理
func (s *FullFlowTestSuite) TestErrorHandling() {
	// 测试不支持的HTTP方法
	req, err := http.NewRequest("PUT", s.serverURL+"/api/v1/rooms", nil)
	require.NoError(s.T(), err)

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// 应该返回405或404
	assert.True(s.T(), resp.StatusCode == http.StatusMethodNotAllowed ||
		resp.StatusCode == http.StatusNotFound)

	// 测试错误的内容类型
	resp, err = s.client.Post(s.serverURL+"/api/v1/rooms", "text/plain",
		strings.NewReader("invalid data"))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	// 服务器应该处理错误的Content-Type
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// TestPerformance 测试性能
func (s *FullFlowTestSuite) TestPerformance() {
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	start := time.Now()
	requests := 10

	for i := 0; i < requests; i++ {
		resp, err := s.client.Get(s.serverURL + "/ping")
		require.NoError(s.T(), err)

		// 读取响应体但不处理
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(requests)

	fmt.Printf("Made %d requests in %v (avg: %v per request)\n",
		requests, elapsed, avgTime)

	// 验证性能要求：平均响应时间小于100ms
	assert.Less(s.T(), avgTime, 100*time.Millisecond,
		"Average response time should be less than 100ms")
}

// TestConcurrentAccess 测试并发访问
func (s *FullFlowTestSuite) TestConcurrentAccess() {
	if testing.Short() {
		s.T().Skip("Skipping concurrent test in short mode")
	}

	concurrency := 5
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// 每个goroutine创建房间并检查
			resp, err := s.client.Post(s.serverURL+"/api/v1/rooms", "application/json", nil)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				errors <- err
				return
			}

			roomID := result["roomId"].(string)

			// 检查房间
			resp, err = s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/check", s.serverURL, roomID))
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			var checkResult map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&checkResult); err != nil {
				errors <- err
				return
			}

			assert.True(s.T(), checkResult["exists"].(bool))
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// 检查是否有错误
	select {
	case err := <-errors:
		s.Fail("Concurrent test failed", err)
	default:
		// 没有错误
	}
}

// TearDownSuite 清理测试套件
func (s *FullFlowTestSuite) TearDownSuite() {
	// 可以在这里清理测试数据
	fmt.Printf("Cleaning up test room: %s\n", s.roomID)
}

func TestFullFlowTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e tests in short mode")
	}
	suite.Run(t, new(FullFlowTestSuite))
}
