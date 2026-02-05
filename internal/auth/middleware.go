package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/vindennt/akasha-showdown-engine/internal/models"
)

type contextKey string

const UserContextKey contextKey = "user"

func (c *Client) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token across Supabase
		user, err := c.AuthClient.WithToken(token).GetUser()
		if err != nil {
			http.Error(w, "Unauthorized: " + err.Error(), http.StatusUnauthorized)
			return
		}

		clientUser := models.User{
			ID:    user.ID.String(),
			Email: user.Email,
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), UserContextKey, clientUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
