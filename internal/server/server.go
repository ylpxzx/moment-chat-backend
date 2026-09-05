package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"moment-chat-backend/internal/config"
	"moment-chat-backend/internal/handler"
	"moment-chat-backend/pkg/redis"
	"moment-chat-backend/pkg/websocket"
)

type Server struct {
	cfg        *config.Config
	router     *gin.Engine
	httpServer *http.Server
	redis      *redis.Client
	wsManager  *websocket.Manager
}

func NewServer(cfg *config.Config) *Server {
	if cfg.Debug {
		gin.SetMode(gin.DebugMode) // 开发模式
	} else {
		gin.SetMode(gin.ReleaseMode) // 生产模式
	}

	// 初始化Redis
	redisClient, err := redis.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// 初始化WebSocket管理器
	wsManager := websocket.NewManager(redisClient.Client)

	// 初始化Gin
	router := gin.Default()

	// 设置CORS中间件
	router.Use(corsMiddleware())

	return &Server{
		cfg:       cfg,
		router:    router,
		redis:     redisClient,
		wsManager: wsManager,
	}
}

func (s *Server) setupRoutes() {
	// 初始化处理器
	roomHandler := handler.NewRoomHandler(s.wsManager)

	// 健康检查端点
	s.router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]interface{}{
			"message":   "pong",
			"service":   "moment-chat-backend",
			"status":    "running",
			"timestamp": time.Now().Unix(),
		})
	})

	// API路由
	api := s.router.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]interface{}{
				"status":    "healthy",
				"version":   "1.0.0",
				"timestamp": time.Now().Unix(),
			})
		})
		api.POST("/rooms", roomHandler.CreateRoom)
		api.GET("/rooms/:roomId/check", roomHandler.CheckRoom)
		api.GET("/rooms/:roomId/info", roomHandler.GetRoomInfo)
		api.GET("/avatars", roomHandler.GetDefaultAvatars)
	}

	// WebSocket路由
	s.router.GET("/ws/rooms/:roomId", func(c *gin.Context) {
		roomID := c.Param("roomId")
		s.wsManager.HandleWebSocket(c.Writer, roomID, c.Request)
	})

	// 静态文件服务（前端）
	s.router.Static("/static", "./static")
	s.router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// 404 处理
	s.router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Not Found",
			"message": "The requested resource was not found.",
		})
	})
}

func (s *Server) Run() error {
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.ServerPort),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %d", s.cfg.ServerPort)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

// CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
