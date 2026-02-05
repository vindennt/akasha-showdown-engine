package auth

import (
	"encoding/json"
	"net/http"

	"github.com/supabase-community/gotrue-go/types"
	"github.com/vindennt/akasha-showdown-engine/internal/models"
)

func (c *Client) Signup(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := c.AuthClient.Signup(types.SignupRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Map to response model
	authRes := models.AuthResponse{
		Message: "Signup & signin successful",
		Session: models.SessionResponse{
			AccessToken:  res.AccessToken,
			RefreshToken: res.RefreshToken,
			ExpiresIn:    res.ExpiresIn,
			TokenType:    res.TokenType,
			User: models.User{
				ID:    res.User.ID.String(),
				Email: res.User.Email,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authRes)
}

func (c *Client) Signin(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := c.AuthClient.SignInWithEmailPassword(req.Email, req.Password)

	if err != nil {
		http.Error(w, "Signin failed: " + err.Error(), http.StatusUnauthorized)
		return
	}

	// Map to res model
	authRes := models.AuthResponse{
		Message: "Signin successful",
		Session: models.SessionResponse{
			AccessToken:  res.AccessToken,
			RefreshToken: res.RefreshToken,
			ExpiresIn:    res.ExpiresIn,
			TokenType:    res.TokenType,
			User: models.User{
				ID:    res.User.ID.String(),
				Email: res.User.Email,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authRes)
}
