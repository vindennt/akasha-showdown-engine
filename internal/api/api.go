package api

import (
	"fmt"
	"net/http"

	"github.com/vindennt/akasha-showdown-engine/internal/config"
)

func RegisterRoutes(mux *http.ServeMux, cfg *config.Config) {
	mux.HandleFunc("GET /{$}", pingHandler)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "pong")
}