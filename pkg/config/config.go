package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl                string
	GoogleClientID       string
	GoogleClientSecret   string
	JWTSecret            string
	PaystackSecret       string
	PaystackChannels     []string
	MinTransactionAmount int64
	Port                 string
	Host                 string
	Env                  string
	AllowedOrigins       []string
	MaxActiveKeys        int
	RedisURL             string
	RedisPassword        string
}

func LoadConfig() Config {
	godotenv.Load()

	paystackChannels := strings.Split(getEnv("PAYSTACK_CHANNELS"), ",")

	return Config{
		DBUrl:                getEnv("DATABASE_URL"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET"),
		JWTSecret:            getEnv("JWT_SECRET"),
		PaystackSecret:       getEnv("PAYSTACK_SECRET"),
		PaystackChannels:     paystackChannels,
		MinTransactionAmount: getEnvAsInt64("MIN_TRANSACTION_AMOUNT"),
		Port:                 getEnv("PORT"),
		Host:                 getEnv("HOST"),
		Env:                  getEnv("ENV"),
		AllowedOrigins:       strings.Split(getEnv("ALLOWED_ORIGINS"), ","),
		MaxActiveKeys:        getEnvAsInt("MAX_ACTIVE_KEYS"),
		RedisURL:             getEnv("REDIS_URL"),
		RedisPassword:        getEnv("REDIS_PASSWORD"),
	}
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic(fmt.Sprintf("%s is required", key))
}

func getEnvAsInt(key string) int {
	valueStr := getEnv(key)

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		panic(fmt.Sprintf("%s must be a valid integer", key))
	}
	return value
}

func getEnvAsInt64(key string) int64 {
	valueStr := getEnv(key)

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("%s must be a valid integer", key))
	}
	return value
}
