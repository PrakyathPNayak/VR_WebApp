package config

import (
    "os"
)

type Config struct {
    ServerAddress string
    MediaDir      string
    StaticDir     string
    DefaultRoom   string
}

func Load() *Config {
    return &Config{
        ServerAddress: getEnv("SERVER_ADDRESS", ":8080"),
        MediaDir:      getEnv("MEDIA_DIR", "media"),
        StaticDir:     getEnv("STATIC_DIR", "static"),
        DefaultRoom:   getEnv("DEFAULT_ROOM", "default"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}