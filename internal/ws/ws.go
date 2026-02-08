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
	"github.com/vindennt/akasha-showdown-engine/internal/middleware"
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

	// TODO: multiple lobby support. create,join,leave,delete lobby UI and endpoints
	lobbiesMutex sync.Mutex
	lobbies      map[string]*Lobby // Map of lobby IDs
	globalLobby  *Lobby            // Mian lobby all clients are connected to

	// Matchmaking queue
	queueMutex       sync.Mutex
	matchmakingQueue []int // subscriber IDs
}

// GameServer Constructor
func NewGameServer(mux *http.ServeMux) *GameServer {
	globalLobby := &Lobby{
		ID:          "global",
		Name:        "Global Lobby",
		subscribers: make(map[int]*Subscriber),
	}

	gs := &GameServer{
		subscriberMessageBuffer: 12,
		publishLimiter:          rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
		logf:                    log.Printf,
		serveMux:                mux,
		lobbies:                 make(map[string]*Lobby),
		globalLobby:             globalLobby,
		matchmakingQueue:        make([]int, 0), // First players in should get priority. Might use a queue window later
	}

	// Add global lobby to lobbies map
	gs.lobbies["global"] = globalLobby

	// Register WebSocket endpoints
	gs.serveMux.HandleFunc("/ws/subscribe", gs.subscribeHandler)
	gs.serveMux.HandleFunc("/ws/publish", gs.publishHandler)

	// Chat and lobby endpoints with CORS support
	gs.serveMux.HandleFunc("/ws/chat", middleware.CORS(gs.chatHandler))
	gs.serveMux.HandleFunc("/ws/lobby/join", middleware.CORS(gs.joinLobbyHandler))
	gs.serveMux.HandleFunc("/ws/queue/join", middleware.CORS(gs.joinQueueHandler))

	return gs
}

// Implement http.Handler interface so it can be an http server handler
// Delegates requests to the appropriate handler
func (gs *GameServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gs.serveMux.ServeHTTP(w, r)
}


// Publish message to all subscribers in global lobby
func (gs *GameServer) publish(msg []byte) {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()

	// Blocks until the rate limiter allows publishing (indefintely with background context)
	gs.publishLimiter.Wait(context.Background())

	// For each subscriber: send message on their channel
	// If the subscriber cannot immediately receive the message,
	// they are considered too slow and closeSlow is called
	for _, s := range gs.globalLobby.subscribers {
		select {
		case s.messc <- msg:
		default:
			go s.closeSlow()
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

// Add single subscriber to global lobby
func (gs *GameServer) addSubscriber(s *Subscriber) {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()
	gs.globalLobby.subscribers[s.ID()] = s
}

// Remove single subscriber from global lobby
func (gs *GameServer) removeSubscriber(s *Subscriber) {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()
	delete(gs.globalLobby.subscribers, s.ID())
}

// GetSubscriber returns a subscriber by ID (O(1) lookup)
// Returns nil if subscriber not found
func (gs *GameServer) GetSubscriber(id int) *Subscriber {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()
	return gs.globalLobby.subscribers[id]
}

// SubscriberCount returns the number of subscribers in global lobby
func (gs *GameServer) SubscriberCount() int {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()
	return len(gs.globalLobby.subscribers)
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

// subscribersList array of all peers in global lobby
func (gs *GameServer) getSubscribers() []Peer {
	gs.globalLobby.mutex.Lock()
	defer gs.globalLobby.mutex.Unlock()

	peers := make([]Peer, 0, len(gs.globalLobby.subscribers))

	for _, s := range gs.globalLobby.subscribers {
		peers = append(peers, Peer{
			Type:  "PEER_JOIN",
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

	// TODO: consider refactoring this to handler?
	// Broadcast lobby join event
	joinEvent := LobbyEvent{
		Type:    "LOBBY_JOIN",
		UserID:  s.ID(),
		LobbyID: "global",
	}
	joinMsg, _ := json.Marshal(joinEvent)
	gs.publish(joinMsg)

	// Broadcast lobby leave event when disconnecting
	defer func() {
		leaveEvent := LobbyEvent{
			Type:    "LOBBY_LEAVE",
			UserID:  s.ID(),
			LobbyID: "global",
		}
		leaveMsg, _ := json.Marshal(leaveEvent)
		gs.publish(leaveMsg)
	}()

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