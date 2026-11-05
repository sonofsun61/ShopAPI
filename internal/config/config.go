package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ConnString      string
	AuthServiceAddr string
	RedisAddr       string
}

func MustLoadConfig() *Config {
	_ = godotenv.Load()
	host := os.Getenv("DATABASE_HOST")
	if host == "" {
		host = "localhost"
	}
	username := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	port := os.Getenv("DATABASE_PORT")
	dbName := os.Getenv("DATABASE_NAME")
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	redisAddr := os.Getenv("REDIS_ADDR")
	if authAddr == "" {
		panic("AUTH_SERVICE_ADDR is not set")
	}
	return &Config{
		ConnString:      fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, dbName),
		AuthServiceAddr: authAddr,
		RedisAddr:       redisAddr,
	}
}
