package db

import (
	"github.com/supabase-community/postgrest-go"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
)

// Client manages Supabase database connections
type Client struct {
	baseURL   string
	anonKey   string
	secretKey string
}

// NewClient creates a new database client from config
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:   cfg.SupabaseURL,
		anonKey:   cfg.SupabaseAnonKey,
		secretKey: cfg.SupabaseSecretKey,
	}
}

// GetUserClient returns a PostgREST client for authenticated user requests
// Uses the provided JWT token for authentication
func (c *Client) GetUserClient(token string) *postgrest.Client {
	restURL := c.baseURL + "/rest/v1"
	
	headers := map[string]string{
		"apikey": c.anonKey,
	}
	
	client := postgrest.NewClient(restURL, "", headers)
	
	if token != "" {
		client.SetAuthToken(token)
	} else {
		client.SetAuthToken(c.anonKey)
	}
	
	return client
}

// GetSystemClient returns a PostgREST client for system-level operations
// Uses the secret key which bypasses Row Level Security policies
func (c *Client) GetSystemClient() *postgrest.Client {
	restURL := c.baseURL + "/rest/v1"
	
	// Use secret key if available, fallback to anon key
	authKey := c.secretKey
	if authKey == "" {
		authKey = c.anonKey
	}
	
	headers := map[string]string{
		"apikey": authKey, // Secret key bypasses RLS
	}
	
	client := postgrest.NewClient(restURL, "", headers)
	client.SetAuthToken(authKey)
	
	return client
}
