package auth

import (
	"github.com/supabase-community/gotrue-go"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
)

type Client struct {
	AuthClient gotrue.Client
}

func NewClient(cfg *config.Config) *Client {
	client := gotrue.New(
		cfg.SupabaseProjectRef,
		cfg.SupabaseAnonKey,
	)
	return &Client{
		AuthClient: client,
	}
}
