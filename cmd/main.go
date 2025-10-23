package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	log.Printf("Starting akasha-showdown-engine server...")

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

// Runs the HTTP server
// Returns any errors
func run() error {
	if len(os.Args) < 2 {
		return errors.New("Error: Provide an address to listen on as the first argument")
	}

	// Create TCP address listener l
	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}
	log.Printf("Now listening on ws://%v", l.Addr())

	// Create HTTP server using gameServer websocket handler
	gs := newGameServer()
	s := &http.Server{
		Handler: gs,
		ReadTimeout: time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

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