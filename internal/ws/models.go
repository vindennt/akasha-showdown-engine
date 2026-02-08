package ws

import (
	"log"
	"sync"
)

var (
	nextSubscriberID   int
	nextSubscriberIDMu sync.Mutex
)

type Peer struct {
	Type  string `json:"type"`
	ID    int    `json:"id"`
	State string `json:"state"`
}

// Lobby represents a chat/game lobby
type Lobby struct {
	ID          string
	Name        string
	subscribers map[int]*Subscriber // Map subscriber ID to subscriber for O(1) lookup
	mutex       sync.Mutex
}

type ChatMessage struct {
	Type      string `json:"type"` // "CHAT_MESSAGE"
	SenderID  int    `json:"sender_id"`
	Message   string `json:"message"`
	LobbyID   string `json:"lobby_id"`
	Timestamp int64  `json:"timestamp"`
}

type LobbyInfo struct {
	Type     string `json:"type"` // "LOBBY_INFO"
	ID       string `json:"id"`
	Name     string `json:"name"`
	NumUsers int    `json:"num_users"`
}

type LobbyEvent struct {
	Type    string `json:"type"` // "LOBBY_JOIN", "LOBBY_LEAVE"
	UserID  int    `json:"user_id"`
	LobbyID string `json:"lobby_id"`
}

// Matchmaking 
type MatchResult struct {
	Type     string `json:"type"` // "MATCH_RESULT"
	WinnerID int    `json:"winner_id"`
	LoserID  int    `json:"loser_id"`
}

// subscriber represents a subscriber
// Each subscriber gets a unique numeric id (starting at 0), a message channel
// and a closeSlow callback.
type Subscriber struct {
	id        int         // unique subscriber id (0-based)
	messc     chan []byte // Channel for incoming messages
	closeSlow func()
}

// newSubscriber allocates a subscriber id (starting at 0) and returns a ready subscriber.
func NewSubscriber(messc chan []byte, closeSlow func()) *Subscriber {
	nextSubscriberIDMu.Lock()
	id := nextSubscriberID
	nextSubscriberID++
	nextSubscriberIDMu.Unlock()

	log.Printf("Assigned new subscriber id=%d", id)

	return &Subscriber{
		id:        id,
		messc:     messc,
		closeSlow: closeSlow,
	}
}

// ID returns the subscriber's id.
func (s *Subscriber) ID() int { return s.id }