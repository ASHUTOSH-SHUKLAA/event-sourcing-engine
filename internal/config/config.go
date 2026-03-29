package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}
}

func GetDBUrl() string {
	return os.Getenv("DATABASE_URL")
}

func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

func GetJWTRefreshSecret() string {
	return os.Getenv("JWT_REFRESH_SECRET")
}
