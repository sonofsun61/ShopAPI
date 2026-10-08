package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ConnString      string
	AuthServiceAddr string
}

func MustLoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		panic("Could not load .env file")
	}
	username := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	port := os.Getenv("DATABASE_PORT")
	dbName := os.Getenv("DATABASE_NAME")
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		panic("AUTH_SERVICE_ADDR is not set")
	}
	return &Config{
		ConnString:      fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", username, password, port, dbName),
		AuthServiceAddr: authAddr,
	}
}
