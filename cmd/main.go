package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vindennt/akasha-showdown-engine/internal/api"
	"github.com/vindennt/akasha-showdown-engine/internal/config"
	"github.com/vindennt/akasha-showdown-engine/internal/ws"
)

func main() {
	log.Printf("Starting akasha-showdown-engine server...")

	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		log.Fatalf("Could not load config: %v", cfgErr)
	}

	err := run(cfg)
	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}

// Runs the HTTP server
// Returns any errors
func run(cfg *config.Config) error {
	// if len(os.Args) < 2 {
	// 	return errors.New("Error: Provide listening address for gameserver as first argument")
	// }

	// Main HTTP request router
	mux := http.NewServeMux()
	ws.NewGameServer(mux)
	api.RegisterRoutes(mux, cfg)
	
	// Create TCP address listener "l"
	addr := fmt.Sprintf(":%s", cfg.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("Server listening on http://localhost%s", addr)
	s := &http.Server{
		Handler: mux,
		ReadTimeout: time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	log.Printf("Now listening on ws://%v", l.Addr())
	
	// Create error channel
	errc := make(chan error, 1)

	// Async goroutine to run the HTTP server
	// Serves HTTP requests onto the listener
	go func() {
		log.Printf("Starting HTTP server...")
		errc <- s.Serve(l)
	}()

	// Create OS signal channel
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	// Wait and listen on the err and sig channels; Logs any received errors/signals
	select {
	case err := <-errc:
		log.Printf("Server error. Failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("Received signal %v. Shutting down server...", sig)
	}

	// Provide context for cleanup time, forcing close after 10 seconds
	// Stop accepting new connections and wait for in-progress requests to finish
	// Free context resources and shutdown the HTTP server cleanly
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}