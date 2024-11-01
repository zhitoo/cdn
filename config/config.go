package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PublicHost   string
	Port         string
	DBUser       string
	DBPassword   string
	DBHost       string
	DBPort       string
	DBName       string
	JWTSecretKey string
}

func initConfig() Config {
	godotenv.Load()
	return Config{
		PublicHost:   getEnv("PUBLIC_HOST", "http://localhost"),
		Port:         getEnv("APP_PORT", "8080"),
		DBUser:       getEnv("DB_USER", "default"),
		DBPassword:   getEnv("DB_PASSWORD", "password"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBName:       getEnv("DB_NAME", "samle_api"),
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
