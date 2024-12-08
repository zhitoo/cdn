package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PublicHost    string
	Port          string
	JWTSecretKey  string
	RedisHost     string
	RedisPort     string
	RedisPassword string
}

func initConfig() Config {
	godotenv.Load()
	return Config{
		PublicHost:    getEnv("PUBLIC_HOST", "http://localhost"),
		Port:          getEnv("APP_PORT", "8080"),
		JWTSecretKey:  getEnv("JWT_SECRET_KEY", "secret"),
		RedisHost:     getEnv("REDIS_HOST", "redis"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var Envs = initConfig()
