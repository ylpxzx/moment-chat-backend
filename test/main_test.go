package test

import (
	"context"
	"fmt"
	"log"
	"moment-chat-backend/internal/config"
	"moment-chat-backend/internal/server"
	"moment-chat-backend/pkg/redis"
	"net/http"
	"os"
	"testing"
	"time"
)

// 定义全局测试变量
var (
	testServerURL string
	testConfig    *config.Config
	testRedis     *redis.Client
)

// TestMain 是所有测试的入口点
func TestMain(m *testing.M) {
	// 设置测试环境
	setupTestEnvironment()

	// 运行测试
	code := m.Run()

	// 清理测试环境
	cleanupTestEnvironment()

	os.Exit(code)
}

// setupTestEnvironment 设置测试环境
func setupTestEnvironment() {
	fmt.Println("Setting up test environment...")

	// 加载测试配置
	testConfig = &config.Config{
		ServerPort: 8081, // 使用不同端口避免冲突
		RedisAddr:  "localhost:6379",
		RedisDB:    1, // 使用不同的DB
		Debug:      true,
	}

	// 初始化Redis测试客户端
	testRedis, err := redis.NewClient(testConfig.RedisAddr, testConfig.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	// 清理测试数据库
	ctx := context.Background()
	testRedis.FlushDB(ctx)

	// 启动测试服务器
	go startTestServer()

	// 等待服务器启动
	time.Sleep(2 * time.Second)
	testServerURL = "http://localhost:8081"

	// 测试连接
	if err := waitForServer(testServerURL); err != nil {
		log.Fatalf("Failed to start test server: %v", err)
	}

	fmt.Println("Test environment setup complete")
}

// cleanupTestEnvironment 清理测试环境
func cleanupTestEnvironment() {
	fmt.Println("Cleaning up test environment...")

	// 清理Redis测试数据库
	if testRedis != nil {
		ctx := context.Background()
		testRedis.FlushDB(ctx)
		testRedis.Close()
	}

	fmt.Println("Test environment cleanup complete")
}

// startTestServer 启动测试服务器
func startTestServer() {
	// 创建服务器实例
	srv := server.NewServer(testConfig)

	// 运行服务器
	if err := srv.Run(); err != nil {
		log.Printf("Test server error: %v", err)
	}
}

// waitForServer 等待服务器启动
func waitForServer(url string) error {
	timeout := time.After(10 * time.Second)
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("server did not start in time")
		case <-tick:
			resp, err := http.Get(url + "/ping")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
		}
	}
}
