package common

import (
	"log"
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

func getEnvOrFallback(key string, fallback string) string {
	value := os.Getenv(key)

	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func getEnvOrPanic(key string) string {
	value := os.Getenv(key)

	if strings.TrimSpace(value) == "" {
		log.Fatalf("Missing %s environment variable", key)
	}

	return value
}

var Env = struct {
	BlogServiceDbString string
	BlogServiceAddr     string
	AdminAddr           string
}{
	BlogServiceDbString: getEnvOrPanic("BLOG_SERVICE_DB_STRING"),
	BlogServiceAddr:     getEnvOrFallback("BLOG_SERVICE_ADDR", "localhost:8000"),
	AdminAddr:           getEnvOrFallback("ADMIN_ADDR", ":8080"),
}
