package config

import (
	"os"
	_ "github.com/joho/godotenv/autoload"
)

// TODO: Add more configuration options as needed
type Config struct {
	// Logs  LogConfig
	// DB    PostgresConfig
	Port  string
	// AllowedOrigin string
}

// type LogConfig struct {
// 	Style string
// 	Level string
// }

// type PostgresConfig struct {
// 	Username string
// 	Password string
// 	URL      string
// 	Port     string
// }

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port: os.Getenv("PORT"),
		// Logs: LogConfig{
		// 	Style: os.Getenv("LOG_STYLE"),
		// 	Level: os.Getenv("LOG_LEVEL"),
		// },
		// DB: PostgresConfig{
		// 	Username: os.Getenv("POSTGRES_USER"),
		// 	Password: os.Getenv("POSTGRES_PWD"),
		// 	URL:      os.Getenv("POSTGRES_URL"),
		// 	Port:     os.Getenv("POSTGRES_PORT"),
		// },
		// AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}

	return cfg, nil
}