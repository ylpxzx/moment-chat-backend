package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort    int
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	Debug         bool
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		println(".env file not loaded:", err.Error())
	}
	println("Loaded PORT from env:", os.Getenv("PORT"))
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	return &Config{
		ServerPort:    port,
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       redisDB,
		Debug:         getEnv("DEBUG", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
