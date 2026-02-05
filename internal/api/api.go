package api

import (
	"net/http"

	"github.com/vindennt/akasha-showdown-engine/internal/auth"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
)

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://localhost:5173")
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	authClient := auth.NewClient(cfg)
	itemHandler := NewItemHandler(cfg)
	// TODO:
	// logHandler := middleware.Logging(corsHandler)
	
	// Health Check
	mux.HandleFunc("/health/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "pong"}`))
	})

	mux.Handle("/auth/signup", corsMiddleware(http.HandlerFunc(authClient.Signup)))
	mux.Handle("/auth/signin", corsMiddleware(http.HandlerFunc(authClient.Signin)))

	mux.Handle("/item/create-item", corsMiddleware(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.CreateItem))))
	mux.Handle("/item/get-item/", corsMiddleware(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.GetItem))))
	mux.Handle("/item/get-items", corsMiddleware(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.ListItems))))
	mux.Handle("/item/update-item/", corsMiddleware(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.UpdateItem))))
	mux.Handle("/item/delete/", corsMiddleware(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.DeleteItem))))
}

