package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         int
	DBDriver     string
	DBHost       string
	DBPort       int
	DBName       string
	DBUser       string
	DBPassword   string
	SyncInterval time.Duration
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func LoadConfig() *Config {
	syncIntervalSeconds := getEnvInt("SYNC_INTERVAL_SECONDS", 5)

	return &Config{
		Port:         getEnvInt("PORT", 8080),
		DBDriver:     getEnv("DB_DRIVER", "postgresql"),
		DBHost:       getEnv("DB_HOST", "127.0.0.1"),
		DBPort:       getEnvInt("DB_PORT", 5432),
		DBName:       getEnv("DB_NAME", "config_manager"),
		DBUser:       getEnv("DB_USER", "root"),
		DBPassword:   getEnv("DB_PASSWORD", "secret_password"),
		SyncInterval: time.Duration(syncIntervalSeconds) * time.Second,
	}
}
