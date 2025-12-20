// package test

// import (
// 	"log"
// 	"net/http"
// 	"strconv"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"

// 	"moment-chat-backend/internal/config"
// )

// func test() {
// 	// 加载配置
// 	cfg := config.Load()

// 	// 根据配置设置Gin模式
// 	if cfg.Debug {
// 		gin.SetMode(gin.DebugMode)
// 	} else {
// 		gin.SetMode(gin.ReleaseMode)
// 	}

// 	// 创建Gin实例
// 	r := gin.Default()

// 	// 添加CORS中间件
// 	r.Use(corsMiddleware())

// 	// 测试路由
// 	r.GET("/ping", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{
// 			"message": "pong",
// 			"service": "moment-chat",
// 			"status":  "running",
// 		})
// 	})

// 	// API v1 路由组
// 	api := r.Group("/api/v1")
// 	{
// 		api.GET("/health", healthCheck)
// 		api.POST("/rooms", createRoom)
// 		api.GET("/rooms/:id/check", checkRoom)
// 	}

// 	// 静态文件服务
// 	r.Static("/static", "./static")
// 	r.GET("/", func(c *gin.Context) {
// 		c.File("./static/index.html")
// 	})

// 	// 启动服务器
// 	addr := ":" + strconv.Itoa(cfg.ServerPort)
// 	log.Printf("Server starting on %s", addr)
// 	if err := r.Run(addr); err != nil {
// 		log.Fatal("Failed to start server:", err)
// 	}
// }

// // CORS中间件
// func corsMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
// 		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
// 		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
// 		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

// 		if c.Request.Method == "OPTIONS" {
// 			c.AbortWithStatus(204)
// 			return
// 		}

// 		c.Next()
// 	}
// }

// // 健康检查
// func healthCheck(c *gin.Context) {
// 	c.JSON(http.StatusOK, gin.H{
// 		"status":    "healthy",
// 		"timestamp": time.Now().Unix(),
// 	})
// }

// // 创建房间
// func createRoom(c *gin.Context) {
// 	// 暂时返回模拟数据
// 	c.JSON(http.StatusOK, gin.H{
// 		"roomId":    uuid.New().String()[:6],
// 		"createdAt": time.Now().Unix(),
// 		"message":   "Room created successfully",
// 	})
// }

// // 检查房间
// func checkRoom(c *gin.Context) {
// 	roomId := c.Param("id")

// 	// 暂时返回模拟数据
// 	c.JSON(http.StatusOK, gin.H{
// 		"exists": true,
// 		"roomId": roomId,
// 	})
// }
