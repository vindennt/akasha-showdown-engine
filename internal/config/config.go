package config

import (
	"os"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

// TODO: Add more configuration options as needed
type Config struct {
	// Logs  LogConfig
	// DB    PostgresConfig
	Port               string
	SupabaseURL        string
	SupabaseProjectRef string
	SupabaseAnonKey    string
	SupabaseSecretKey  string // Secret key for server-side operations (replaces legacy service_role)
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
	port := os.Getenv("PORT")
	anon_key := os.Getenv("SUPABASE_KEY")

	supabaseURL := os.Getenv("SUPABASE_URL")
	SupabaseSecretKey := os.Getenv("SUPABASE_SECRET_KEY")``

	// Extract project ref key (strip protocol first)
	projectRef := supabaseURL
	// Remove https:// or http:// prefix
	projectRef = strings.TrimPrefix(projectRef, "https://")
	projectRef = strings.TrimPrefix(projectRef, "http://")
	if idx := strings.Index(projectRef, ".supabase.co"); idx != -1 {
		projectRef = projectRef[:idx]
	}

	cfg := &Config{
		Port:               port,
		SupabaseURL:        supabaseURL,
		SupabaseProjectRef: projectRef,
		SupabaseAnonKey:    anon_key,
		SupabaseSecretKey:  SupabaseSecretKey,
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