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
}

func LoadConfig() Config {
	godotenv.Load()

	paystackChannels := strings.Split(getEnv("PAYSTACK_CHANNELS"), ",")

	minAmountStr := getEnv("MIN_TRANSACTION_AMOUNT")
	minAmount, err := strconv.ParseInt(minAmountStr, 10, 64)
	if err != nil {
		panic("MIN_TRANSACTION_AMOUNT must be a valid integer")
	}

	return Config{
		DBUrl:                getEnv("DATABASE_URL"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET"),
		JWTSecret:            getEnv("JWT_SECRET"),
		PaystackSecret:       getEnv("PAYSTACK_SECRET"),
		PaystackChannels:     paystackChannels,
		MinTransactionAmount: minAmount,
		Port:                 getEnv("PORT"),
		Host:                 getEnv("HOST"),
		Env:                  getEnv("ENV"),
		AllowedOrigins:       strings.Split(getEnv("ALLOWED_ORIGINS"), ","),
	}
}

func getEnv(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	panic(fmt.Sprintf("%s is required", key))
}
