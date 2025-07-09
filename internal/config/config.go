package config

import (
	"os"
)

type Config struct {
	ServerAddress   string
	MediaDir        string
	StaticDir       string
	DefaultRoom     string
	DefaultFilePath string
}

func Load() *Config {
	return &Config{
		ServerAddress:   getEnv("SERVER_ADDRESS", "0.0.0.0:8000"),
		MediaDir:        getEnv("MEDIA_DIR", "media"),
		StaticDir:       getEnv("STATIC_DIR", "static"),
		DefaultRoom:     getEnv("DEFAULT_ROOM", "default"),
		DefaultFilePath: getEnv("filePath", "execs/VRenv(raylib).exe"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
