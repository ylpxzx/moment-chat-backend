package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

// NewClient 创建并返回一个 Redis 客户端实例。
//
// 该函数会建立与 Redis 服务器的连接,并通过 Ping 命令验证连接是否成功。
// 如果连接失败,程序将直接 panic。
//
// 参数:
//   - addr: Redis 服务器地址,格式为 "host:port",例如 "localhost:6379"
//   - password: Redis 连接密码；无密码时传空字符串
//   - db: 要选择的 Redis 数据库索引号,范围通常为 0-15
//
// 返回值:
//   - *Client: 初始化后的 Redis 客户端指针,可用于执行 Redis 操作
//
// 注意:
//   - 连接测试超时时间设置为 5 秒
//   - 若连接失败会触发 panic,调用方应做好错误处理准备
func NewClient(addr, password string, db int) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{client}, nil
}
