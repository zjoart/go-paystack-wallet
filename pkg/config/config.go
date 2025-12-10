package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl              string
	GoogleClientID     string
	GoogleClientSecret string
	JWTSecret          string
	Port               string
	Host               string
	Env                string
	AllowedOrigins     []string
}

func LoadConfig() Config {
	godotenv.Load()

	return Config{
		DBUrl:              getEnv("DATABASE_URL"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET"),
		JWTSecret:          getEnv("JWT_SECRET"),
		Port:               getEnv("PORT"),
		Host:               getEnv("HOST"),
		Env:                getEnv("ENV"),
		AllowedOrigins:     strings.Split(getEnv("ALLOWED_ORIGINS"), ","),
	}
}

func getEnv(key string) string {

	if value := os.Getenv(key); value != "" {
		return value
	}

	panic(fmt.Sprintf("%s is required", key))
}
