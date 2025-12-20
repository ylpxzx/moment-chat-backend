package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APITestSuite API测试套件
type APITestSuite struct {
	suite.Suite
	serverURL string
	client    *http.Client
}

// SetupSuite 测试套件设置
func (s *APITestSuite) SetupSuite() {
	s.serverURL = "http://localhost:8080"
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}

	// 等待服务器就绪
	s.waitForServer()
}

// waitForServer 等待服务器启动并可用
func (s *APITestSuite) waitForServer() {
	// 最多重试10次，每次间隔500毫秒
	for i := 0; i < 10; i++ {
		// 尝试请求 /ping 健康检查接口
		resp, err := http.Get(s.serverURL + "/ping")
		// 如果请求成功且返回200，说明服务已启动
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		// 否则等待一段时间后重试
		time.Sleep(500 * time.Millisecond)
	}
	// 超过重试次数仍未启动则测试失败
	s.FailNow("Server did not start in time")
}

// TestHealthEndpoints 测试健康检查端点
func (s *APITestSuite) TestHealthEndpoints() {
	tests := []struct {
		name     string
		endpoint string
		expected map[string]interface{}
	}{
		{
			name:     "Test /ping endpoint",
			endpoint: "/ping",
			expected: map[string]interface{}{
				"message": "pong",
				"service": "moment-chat-backend",
				"status":  "running",
			},
		},
		{
			name:     "Test /api/v1/health endpoint",
			endpoint: "/api/v1/health",
			expected: map[string]interface{}{
				"status":  "healthy",
				"version": "1.0.0",
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp, err := s.client.Get(s.serverURL + tt.endpoint)
			require.NoError(s.T(), err)
			defer resp.Body.Close()
			// 打印resp的值内容
			assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(s.T(), err)

			for key, expectedValue := range tt.expected {
				assert.Equal(s.T(), expectedValue, result[key])
			}

			// 检查时间戳是否存在
			assert.Contains(s.T(), result, "timestamp")
		})
	}
}

// TestRoomCreationFlow 测试房间创建流程
func (s *APITestSuite) TestRoomCreationFlow() {
	// 1. 创建房间
	resp, err := s.client.Post(s.serverURL+"/api/v1/rooms", "application/json", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	require.NoError(s.T(), err)
	fmt.Printf("createResp: %+v\n", createResp)
	roomID, ok := createResp["roomId"].(string)
	require.True(s.T(), ok, "roomId should be a string")
	assert.Len(s.T(), roomID, 6, "roomId should be 6 characters")
	assert.Equal(s.T(), "Room created successfully", createResp["message"])

	// 2. 检查房间是否存在
	resp, err = s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/check", s.serverURL, roomID))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var checkResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&checkResp)
	require.NoError(s.T(), err)
	fmt.Printf("checkResp: %+v\n", checkResp)
	assert.Equal(s.T(), roomID, checkResp["roomId"])
	assert.True(s.T(), checkResp["exists"].(bool))

	// 3. 获取房间信息
	resp, err = s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/info", s.serverURL, roomID))
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var infoResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&infoResp)
	require.NoError(s.T(), err)
	fmt.Printf("infoResp: %+v\n", infoResp)
	assert.Equal(s.T(), roomID, infoResp["roomId"])
	assert.Equal(s.T(), float64(0), infoResp["userCount"])
}

// TestInvalidRoomID 测试无效房间ID
func (s *APITestSuite) TestInvalidRoomID() {
	invalidIDs := []string{
		"123",     // 太短
		"1234567", // 太长
		"123-45",  // 包含特殊字符
		"",        // 空字符串
		"123456",
	}

	for _, roomID := range invalidIDs {
		s.Run(fmt.Sprintf("Test invalid room ID: %s", roomID), func() {
			resp, err := s.client.Get(fmt.Sprintf("%s/api/v1/rooms/%s/check", s.serverURL, roomID))
			require.NoError(s.T(), err)
			defer resp.Body.Close()

			// 包含特殊字符时应该返回400
			if strings.ContainsAny(roomID, "-!@#$%^&*()_+=[]{}|;:'\",.<>/?") {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
				var errorResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(s.T(), err)
				assert.Contains(s.T(), errorResp, "error")
				return
			}

			// 太短的ID应该返回400
			if len(roomID) < 6 {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)

				var errorResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(s.T(), err)
				assert.Contains(s.T(), errorResp, "error")
			}
			if len(roomID) > 6 {
				assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
			}

			if len(roomID) == 6 {
				assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
			}
		})
	}
}

// TestDefaultAvatars 测试获取默认头像
func (s *APITestSuite) TestDefaultAvatars() {
	resp, err := s.client.Get(s.serverURL + "/api/v1/avatars")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var respData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	require.NoError(s.T(), err)

	avatars, ok := respData["avatars"].([]interface{})
	require.True(s.T(), ok, "avatars should be an array")
	assert.Greater(s.T(), len(avatars), 0, "should have at least one avatar")

	// 检查每个头像URL格式
	for _, avatar := range avatars {
		avatarStr, ok := avatar.(string)
		assert.True(s.T(), ok, "avatar should be a string")
		assert.True(s.T(), strings.HasPrefix(avatarStr, "http"), "avatar should be a URL")
	}
}

// TestCORSHeaders 测试CORS头
func (s *APITestSuite) TestCORSHeaders() {
	// 测试OPTIONS请求
	req, err := http.NewRequest("OPTIONS", s.serverURL+"/api/v1/rooms", nil)
	require.NoError(s.T(), err)

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)

	// 检查CORS头
	assert.Equal(s.T(), "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(s.T(), "true", resp.Header.Get("Access-Control-Allow-Credentials"))

	// 测试普通请求也有CORS头
	resp, err = s.client.Get(s.serverURL + "/ping")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), "*", resp.Header.Get("Access-Control-Allow-Origin"))
}

// TestNotFoundHandler 测试404处理
func (s *APITestSuite) TestNotFoundHandler() {
	resp, err := s.client.Get(s.serverURL + "/non-existent-endpoint")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)

	var respData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	require.NoError(s.T(), err)

	assert.Contains(s.T(), respData, "error")
	assert.Contains(s.T(), respData["error"], "Not Found")
}

// 运行测试套件
func TestAPITestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(APITestSuite))
}
