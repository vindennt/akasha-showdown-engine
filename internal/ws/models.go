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