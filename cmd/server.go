package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/coder/websocket"
)

// TODO: make this a more proper enum or type 
colors := []string{"red", "green", "blue", "orange", "purple"}

type Peer struct {
	Type  string
	ID    int
	State string
	Color string
}

// NEW SUBSCRIBER REPRESENTATION
// TODO: ensure this gives a new id to each subscriber
// subscriber represents a subscrber
// Messages sent on message channel messc
// closeSlow called if client cannot keep up with the message rate
var (
	nextSubscriberID   int
	nextSubscriberIDMu sync.Mutex
)

// subscriber represents a subscriber
// Each subscriber gets a unique numeric id (starting at 0), a message channel
// and a closeSlow callback.
type subscriber struct {
	id        int         // unique subscriber id (0-based)
	messc     chan []byte // Channel for incoming messages
	closeSlow func()
}

// newSubscriber allocates a subscriber id (starting at 0) and returns a ready subscriber.
func newSubscriber(messc chan []byte, closeSlow func()) *subscriber {
	nextSubscriberIDMu.Lock()
	id := nextSubscriberID
	nextSubscriberID++
	nextSubscriberIDMu.Unlock()

	log.Printf("assigned new subscriber id=%d", id)

	return &subscriber{
		id:        id,
		messc:     messc,
		closeSlow: closeSlow,
	}
}

// Old subscriber implementation
// type subscriber struct {
// 	messc	chan []byte // Channel for incoming messages
// 	closeSlow	func()
// }

// ID returns the subscriber's id.
func (s *subscriber) ID() int { return s.id }

type gameServer struct {
	// Controls the message queue's window size
	// Messages exceeding the window get dropped
	subscriberMessageBuffer int

	// Controls the rate limit to a publish endpoint per client
	// Default: 1 every 100ms, burst capacity of 8
	publishLimiter *rate.Limiter

	// Sets logger to the default log.Printf
	// Add a custom logger here
	logf func(format string, v ...any)

	// Router for endpoints to corresponding handlers e.g. /chat
	serveMux http.ServeMux

	// Mutex to ensure thread-safe (goroutine-safe) access to subscribers
	// Prevents race conditions
	subscribersMutex sync.Mutex
	
	// Map containing pointers to subscribers
	subscribers   map[*subscriber]struct{}
}

// gameServer Constructor
func newGameServer() *gameServer {
	gs := &gameServer{
		subscriberMessageBuffer: 12,
		publishLimiter: 		rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
		logf:					log.Printf,
		subscribers:			make(map[*subscriber]struct{}),
	}

	// Serves HTTP static file from the current directory
	gs.serveMux.Handle("/", http.FileServer(http.Dir("cmd/static_chat")))
	gs.serveMux.HandleFunc("/subscribe", gs.subscribeHandler)
	gs.serveMux.HandleFunc("/publish", gs.publishHandler)

	return gs
}

// Implement http.Handler interface so it can be an http server handler
// Delegates requests to the appropriate handler
func (gs *gameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gs.serveMux.ServeHTTP(w, r)
}


// Publish message to all subscribers in subscribers map
func (gs *gameServer) publish(msg []byte) {
	// Lock mutex to ensure thread-safe access to subscribers map
	// Unlock mutex after function ends no matter what
	// Necessary because of panics or early returns during
	// waiting or iterations
	gs.subscribersMutex.Lock()
	defer gs.subscribersMutex.Unlock()

	// Blocks until the rate limiter allows publishing
	gs.publishLimiter.Wait(context.Background())

	// For each subscriber sub: send message msg on their channel messc
	// If the subscriber cannot immediately receive the message,
	// they are considered too slow and closeSlow is called
	for sub := range gs.subscribers {
		select {
		case sub.messc <- msg:
		default:
			sub.closeSlow()
		}
	}
}

func (gs *gameServer) publishHandler(w http.ResponseWriter, r *http.Request) {
	// Return Method Not Allowed if not POST
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// Receive request and and limit body size to 8KB. Adjust as needed.
	body := http.MaxBytesReader(w, r.Body, 8192)
	msg, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	gs.publish(msg)

	w.WriteHeader(http.StatusAccepted)
}

// Adds sub to subscribers map
func (gs *gameServer) addSubscriber(sub *subscriber) {
	gs.subscribersMutex.Lock()
	// Manual unlock is okay too but less safe in case of map error
	defer gs.subscribersMutex.Unlock()

	// Add subscriber to subscribers map
	// struct is empty to be 0 bytes, the key is the sub pointer
	gs.subscribers[sub] = struct{}{}
}

// broadcastAll sends msg to all subscribers.
func (gs *gameServer) broadcastAll(msg []byte) {
	gs.subscribersMutex.Lock()
	defer gs.subscribersMutex.Unlock()

	for sub := range gs.subscribers {
		select {
		case sub.messc <- msg:
		default:
			sub.closeSlow()
		}
	}
}

// Removes sub from subscribers map
func (gs *gameServer) removeSubscriber(sub *subscriber) {
	gs.subscribersMutex.Lock()
	defer gs.subscribersMutex.Unlock()

	// Remove subscriber from subscribers map
	delete(gs.subscribers, sub)
}

// Writes msg to the WebSocket connection conn
// Returns an error if the context is canceled or if writing fails
// uses a timeout to prevent slow clients from blocking indefinitely
// cleans up the context after func ends
func writeTimeout(ctx context.Context, timeout time.Duration, conn *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return conn.Write(ctx, websocket.MessageText, msg)
}

// Accepts WebSocket connections and subscribes clients
// to incoming messages
// Handles errors and whether client or server canceled connection already
func (gs *gameServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := gs.subscribe(w, r)
	// Check if context is canceled already by server or client
	if errors.Is(err, context.Canceled) {
		gs.logf("Subscription canceled: %v", err)
		return
	}

	if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
		gs.logf("WebSocket connection closed: %v", err)
		return
	}

	if err != nil {
		gs.logf("Failed to subscribe: %v", err)
		return
	}
}

// Subscribes the given WebSocket to all broadcasted messages
// Creates a subscriber with a message channel and registers them.
// Listens for all messages and writes them to the WebSocket
// Uses CloseRead to keep reading from the connection for process
// control messages, and for deciding to cancel context.
// If context ctx is canceled or error occurs, returns and deletes the subscription
func (gs *gameServer) subscribe(w http.ResponseWriter, r *http.Request) error {
	var mutex sync.Mutex
	var conn *websocket.Conn
	var closed bool

	// Initialize subscriber with a unique id
	sub := newSubscriber(make(chan []byte, gs.subscriberMessageBuffer), func() {
		// Using mutex ensures wrong sub isnt set to closed
		mutex.Lock()
		defer mutex.Unlock()

		closed = true

		if conn != nil {
			conn.Close(websocket.StatusPolicyViolation, "Connection is too slow to keep up with messages")
		}
	})

	// Adding and removing subscribers each handle mutex on their own, so we don't need to lock here
	gs.addSubscriber(sub)
	defer gs.removeSubscriber(sub) // Ensure subscriber is removed when function ends

	// Websocket options
	// TODO: Do not allow insecure skip verify
	opts := websocket.AcceptOptions{
		// OriginPatterns: []string{"localhost:5173"},
		InsecureSkipVerify: true,
	}

	// Accept WebSocket connection with options applied
	connRes, err := websocket.Accept(w, r, &opts)
	if err != nil {
		return err
	}

	// Critical Section: Possible race condition accesssing conn and closed
	// closeSlow could have set closed to true from a goroutine from e.g. publish
	// To avoid setting conn to a closed connection, lock the mutex
	// and check the condition of closed
	mutex.Lock()
	if closed {
		mutex.Unlock()
		return net.ErrClosed
	}
	conn = connRes
	mutex.Unlock()
	defer conn.CloseNow() // Ensures connection is closed when function ends

	// Send welcome message with assigned id as JSON so clients can decode it
	welcome := struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	}{Type: "welcome", ID: sub.ID()}
	b, jerr := json.Marshal(welcome)
	if jerr == nil {
		// Try to enqueue the message; if buffer is full, try a direct write
		select {
		case sub.messc <- b:
		default:
			// Fallback: attempt direct write to ensure client receives id
			_ = writeTimeout(context.Background(), time.Second*5, conn, b)
		}
	} else {
		gs.logf("failed to marshal welcome JSON: %v", jerr)
	}

	// TODO: clean up this broadcast
	// Broadcast peerJoin to existing subscribers (excluding the new one)
	// Set a default state and pick a color from a small palette
	peerJoin := struct {
		Type  string `json:"type"`
		ID    int    `json:"id"`
		State string `json:"state"`
		Color string `json:"color"`
	}{
		Type: "PEER_JOIN",
		ID:   sub.ID(),
		State: "---",
		Color: colors[sub.ID()%len(colors)], // TODO: fix/clear this
	}

	pj, pjerr := json.Marshal(peerJoin)

	// Check for joining errors
	if pjerr == nil {
		gs.broadcastAll(pj)
	} else {
		gs.logf("failed to marshal peerJoin JSON: %v", pjerr)
	}
	// TODO: new code ends here

	// Context that is canceled when WebSocket's read connection is closed
	// Ensures the loop below stops when client stops reading
	ctx := conn.CloseRead(context.Background())

	// While loop
	// Listens for messages arriving, with timeout
	// Listens for cancellation of the context (closed connection)
	for {
		select {
			case msg := <-sub.messc:
				// 5 second timeout for writing messages
				err := writeTimeout(ctx, time.Second*5, conn, msg)
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return ctx.Err()
		}
	}
}