package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PublicHost   string
	Port         string
	JWTSecretKey string
}

func initConfig() Config {
	godotenv.Load()
	return Config{
		PublicHost:   getEnv("PUBLIC_HOST", "http://localhost"),
		Port:         getEnv("APP_PORT", "8080"),
		JWTSecretKey: getEnv("JWT_SECRET_KEY", "secret"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var Envs = initConfig()
