package api

import (
	"net/http"

	"github.com/vindennt/akasha-showdown-engine/internal/auth"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
	"github.com/vindennt/akasha-showdown-engine/internal/middleware"
)


func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	authClient := auth.NewClient(cfg)
	itemHandler := NewItemHandler(cfg)
	
	// Health Check
	mux.HandleFunc("/health/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "pong"}`))
	})

	mux.Handle("/auth/signup", middleware.CORSHandler(http.HandlerFunc(authClient.Signup)))
	mux.Handle("/auth/signin", middleware.CORSHandler(http.HandlerFunc(authClient.Signin)))

	mux.Handle("/item/create-item", middleware.CORSHandler(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.CreateItem))))
	mux.Handle("/item/get-item/", middleware.CORSHandler(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.GetItem))))
	mux.Handle("/item/get-items", middleware.CORSHandler(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.ListItems))))
	mux.Handle("/item/update-item/", middleware.CORSHandler(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.UpdateItem))))
	mux.Handle("/item/delete/", middleware.CORSHandler(authClient.AuthMiddleware(http.HandlerFunc(itemHandler.DeleteItem))))
}
