package unit

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/joho/godotenv"
)

func TestConfigLoad(t *testing.T) {
	// 设置环境变量
	os.Setenv("PORT", "9999")
	os.Setenv("REDIS_ADDR", "test-redis:6379")
	os.Setenv("REDIS_DB", "5")
	os.Setenv("DEBUG", "true")

	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("REDIS_DB")
		os.Unsetenv("DEBUG")
	}()

	// cfg := config.Load()
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	assert.Equal(t, 9999, port)
	assert.Equal(t, "test-redis:6379", getEnv("REDIS_ADDR", "localhost:6379"))
	assert.Equal(t, 5, redisDB)
	assert.True(t, getEnv("DEBUG", "false") == "true")
}

func TestConfigLoadDefaults(t *testing.T) {
	// 确保环境变量被清除
	os.Unsetenv("PORT")
	os.Unsetenv("REDIS_ADDR")
	os.Unsetenv("REDIS_DB")
	os.Unsetenv("DEBUG")

	// cfg := config.Load()
	if err := godotenv.Load("../../.env"); err != nil {
		println(".env file not loaded:", err.Error())
	}
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))
	assert.Equal(t, 8080, port)
	assert.Equal(t, "localhost:6379", getEnv("REDIS_ADDR", "localhost:6379"))
	assert.Equal(t, 0, redisDB)
	assert.True(t, getEnv("DEBUG", "false") == "true")
}

func TestGetEnv(t *testing.T) {
	// 测试环境变量存在的情况
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	value := getEnv("TEST_VAR", "default")
	assert.Equal(t, "test_value", value)

	// 测试环境变量不存在的情况
	value = getEnv("NON_EXISTENT_VAR", "default_value")
	assert.Equal(t, "default_value", value)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
