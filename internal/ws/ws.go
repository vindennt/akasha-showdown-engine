package ws

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

type GameServer struct {
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
	serveMux *http.ServeMux

	// Mutex to ensure thread-safe (goroutine-safe) access to subscribers
	// Prevents race conditions
	subscribersMutex sync.Mutex
	
	// Map containing pointers to subscribers
	subscribers   map[*Subscriber]struct{}
}

// GameServer Constructor
func NewGameServer(mux *http.ServeMux) *GameServer {
	gs := &GameServer{
		subscriberMessageBuffer: 12,
		publishLimiter: 		rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
		logf:					log.Printf,
		subscribers:			make(map[*Subscriber]struct{}),
		serveMux: 				mux,
	}

	// Serves HTTP static file from the current directory
	// gs.serveMux.HandleFunc("/subscribe", gs.subscribeHandler)
	// gs.serveMux.HandleFunc("/publish", gs.publishHandler)
	gs.serveMux.HandleFunc("/ws/subscribe", gs.subscribeHandler)
	gs.serveMux.HandleFunc("/ws/publish", gs.publishHandler)

	return gs
}

// Implement http.Handler interface so it can be an http server handler
// Delegates requests to the appropriate handler
func (gs *GameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gs.serveMux.ServeHTTP(w, r)
}


// Publish message to all subscribers in subscribers map
func (gs *GameServer) publish(msg []byte) {
	// Lock mutex to ensure thread-safe access to subscribers map
	// Unlock mutex after function ends no matter what
	// Necessary because of panics or early returns during
	// waiting or iterations
	gs.subscribersMutex.Lock()
	defer gs.subscribersMutex.Unlock()

	// Blocks until the rate limiter allows publishing (indefintely with background context)
	gs.publishLimiter.Wait(context.Background())

	// For each subscriber s: send message msg on their channel messc
	// If the subscriber cannot immediately receive the message,
	// they are considered too slow and closeSlow is called
	for s := range gs.subscribers {
		select {
		case s.messc <- msg:
		default:
			s.closeSlow()
		}
	}
}

func (gs *GameServer) publishHandler(w http.ResponseWriter, r *http.Request) {
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

// Adds s to subscribers map
func (gs *GameServer) addSubscriber(s *Subscriber) {
	gs.subscribersMutex.Lock()
	// Manual unlock is okay too but less safe in case of map error
	defer gs.subscribersMutex.Unlock()

	// Add subscriber to subscribers map
	// struct is empty to be 0 bytes, the key is the sub pointer
	gs.subscribers[s] = struct{}{}
}

// Removes s from subscribers map
func (gs *GameServer) removeSubscriber(s *Subscriber) {
	gs.subscribersMutex.Lock()
	defer gs.subscribersMutex.Unlock()

	// Remove subscriber from subscribers map
	delete(gs.subscribers, s)
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
func (gs *GameServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
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

// subscribersList array of all peers
func (gs *GameServer) getSubscribers() []Peer {
    gs.subscribersMutex.Lock()
    defer gs.subscribersMutex.Unlock()

    peers := make([]Peer, 0, len(gs.subscribers))

    for s := range gs.subscribers {
        peers = append(peers, Peer{
			Type: "PEER_JOIN",
            ID:    s.ID(),
            State: "Joined", // TODO: default
        })
    }

    return peers
}

// Subscribes the given WebSocket to all broadcasted messages
// Creates a subscriber with a message channel and registers them.
// Listens for all messages and writes them to the WebSocket
// Uses CloseRead to keep reading from the connection for process
// control messages, and for deciding to cancel context.
// If context ctx is canceled or error occurs, returns and deletes the subscription
func (gs *GameServer) subscribe(w http.ResponseWriter, r *http.Request) error {
	var mutex sync.Mutex
	var conn *websocket.Conn
	var closed bool

	// Initialize subscriber with a unique id
	s := NewSubscriber(make(chan []byte, gs.subscriberMessageBuffer), func() {
		// Using mutex ensures wrong sub isnt set to closed
		mutex.Lock()
		defer mutex.Unlock()

		closed = true

		if conn != nil {
			conn.Close(websocket.StatusPolicyViolation, "Connection is too slow to keep up with messages")
		}
	})

	// Deferred leave handling
	defer func() {
		// Handle leaving
		peerLeave := Peer{
			Type: "PEER_LEAVE",
			ID:   s.ID(),
			State: "Left",
		}

		pl, _ := json.Marshal(peerLeave) // Turn into []byte
		gs.publish(pl)
	}()

	// Adding and removing subscribers each handle mutex on their own, so we don't need to lock here
	gs.addSubscriber(s)
	defer gs.removeSubscriber(s) // Ensure subscriber is removed when function ends

	// Websocket options
	// TODO: Do not allow insecure skip verify, 
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
		Type string  `json:"type"`
		ID   int     `json:"id"`
		Peers []Peer `json:"peers"`

	}{
		Type: "WELCOME",
		ID: s.ID(),
		Peers: gs.getSubscribers(), 
	}
	
	wj, wjerr := json.Marshal(welcome)
	if wjerr == nil {
		// Try to enqueue the message; if buffer is full, try a direct write
		select {
		case s.messc <- wj:
		default:
			// Fallback: attempt direct write to ensure client receives id
			_ = writeTimeout(context.Background(), time.Second*5, conn, wj)
		}
	} else {
		gs.logf("failed to marshal welcome JSON: %v", wjerr)
	}

	// Add this new subscriber
	// Broadcast peerJoin to existing subscribers except the new one
	// Set a default start state 
	peerJoin := Peer{
		Type: "PEER_JOIN",
		ID:   s.ID(),
		State: "Joined", // Default start state
	}

	pj, _ := json.Marshal(peerJoin) // Turn into []byte
	gs.publish(pj)

	// Init Context that is canceled when WebSocket's read connection is closed
	// Ensures the loop below stops when client stops reading
	ctx := conn.CloseRead(context.Background()) // TODO: Disable this tio enable client to send messasges back (currently is read only)

	// While loop
	// Listens for messages arriving, with timeout
	// Listens for cancellation of the context (closed connection)
	for {
		select {
			case msg := <-s.messc:
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