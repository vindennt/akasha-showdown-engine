package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/coder/websocket"
)

// gameServer is the WebSocket server for akasha-showdown
type gameServer struct {
	// logf controls where llogs are sent
	logf func(format string, v ...any)
}

func (s gameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrades HTTP connection r into a WebSocket connection
	s.logf("New connection request from %s", r.RemoteAddr)
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		s.logf("Failed to accept WebSocket connection: %v", err)
		return
	}
	defer conn.CloseNow()

	s.logf("WebSocket connection established with %s", r.RemoteAddr)

	// Validates subprotocol
	if conn.Subprotocol() != "echo" {
		conn.Close(websocket.StatusPolicyViolation, "Unsupported subprotocol")
		return
	}

	// Send conn ack to client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// get a writer for connection, setts err if context exceeded
	wr, err := conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		s.logf("Failed to send ack to %s: %v", r.RemoteAddr, err)
		return
	}
	wr.Write([]byte("Connection successful"))
	wr.Close()

	// 1 token replenished per second
	// 10 token burst capacity
	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)

	// While loop for handling incoming messages and errors
	for {
		err = echo(conn, l)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			s.logf("WebSocket connection closed normally")
			return
		}
		if err != nil {
			s.logf("WebSocket error: %v", err)
			return
		}
	}
}

func echo(conn *websocket.Conn, l *rate.Limiter) error {
	// Set a 10 second timeout if no messages are received
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := l.Wait(ctx)
	if err != nil {
		return err
	}

	typ, r, err := conn.Reader(ctx)
	if err != nil {
		return err
	}

	w, err := conn.Writer(ctx, typ)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, r)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	err = w.Close()
	return err
}
