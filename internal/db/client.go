package db

import (
	"github.com/supabase-community/postgrest-go"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
)

type Client struct {
	baseURL   string
	anonKey   string
	secretKey string
}

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
	
	// Fallback
	if token != "" {
		client.SetAuthToken(token)
	} else {
		client.SetAuthToken(c.anonKey)
	}
	
	return client
}

// GetSystemClient returns a PostgREST client for system-level operations
// Uses secret key for non client side ops
func (c *Client) GetSystemClient() *postgrest.Client {
	restURL := c.baseURL + "/rest/v1"
	
	authKey := c.secretKey
	headers := map[string]string{
		"apikey": authKey, 
	}
	
	client := postgrest.NewClient(restURL, "", headers)
	client.SetAuthToken(authKey)
	
	return client
}
