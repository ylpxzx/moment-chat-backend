package unit

import (
	"context"
	"testing"
	"time"

	"moment-chat-backend/pkg/redis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis test in short mode")
	}

	// 使用测试Redis实例
	client, err := redis.NewClient("localhost:6379", "794859685", 1)
	require.NoError(t, err, "Failed to create Redis client")
	defer func() {
		client.FlushDB(context.Background())
		client.Close()
	}()

	ctx := context.Background()

	// 测试基本操作
	err = client.Set(ctx, "test:key", "test_value", 10*time.Second).Err()
	require.NoError(t, err)

	val, err := client.Get(ctx, "test:key").Result()
	// t.Logf("获取redis的key值: %v", val)
	require.NoError(t, err)
	assert.Equal(t, "test_value", val)

	// 测试过期
	err = client.Set(ctx, "expiring:key", "value", 1*time.Second).Err()
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	val, err = client.Get(ctx, "expiring:key").Result()
	assert.Error(t, err)
	assert.Empty(t, val)
}

func TestRedisPing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis test in short mode")
	}

	client, err := redis.NewClient("localhost:6379", "794859685", 1)
	require.NoError(t, err, "Failed to create Redis client")
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.Ping(ctx).Err()
	require.NoError(t, err, "Redis should be reachable")
}
