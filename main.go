package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"moment-chat-backend/internal/config"
	"moment-chat-backend/internal/server"
)

func main() {
	// 加载配置
	cfg := config.Load()
	log.Printf("Loaded config: %+v\n", cfg)
	// 创建服务
	srv := server.NewServer(cfg)

	// 启动服务
	go func() {
		if err := srv.Run(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	srv.Shutdown()
}
