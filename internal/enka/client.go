package enka

import (
	"context"
	"time"

	"github.com/kirinyoku/enkanetwork-go/client/genshin"
)

type Client struct {
	api *genshin.Client
}

func NewClient(userAgent string) *Client {
	// TODO: add caching for data
	api := genshin.NewClient(nil, nil, userAgent)
	return &Client{
		api: api,
	}
}

func (c *Client) GetPlayerInfo(ctx context.Context, uid string) (*genshin.Profile, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	return c.api.GetProfile(ctx, uid)
}
