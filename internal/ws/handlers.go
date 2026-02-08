package ws

import (
	"context"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// publishes a message to all subscribers in a specific lobby
func (gs *GameServer) publishToLobby(lobbyID string, msg []byte) {
	gs.lobbiesMutex.Lock()
	lobby, exists := gs.lobbies[lobbyID]
	gs.lobbiesMutex.Unlock()

	if !exists {
		gs.logf("[ERROR] Lobby '%s' does not exist", lobbyID)
		return
	}

	lobby.mutex.Lock()
	subscriberCount := len(lobby.subscribers)
	gs.logf("[PUBLISH] Publishing message to lobby '%s' with %d subscribers", lobbyID, subscriberCount)
	lobby.mutex.Unlock()

	// Rate limit
	gs.publishLimiter.Wait(context.Background())

	lobby.mutex.Lock()
	defer lobby.mutex.Unlock()

	sentCount := 0
	for _, s := range lobby.subscribers {
		select {
		case s.messc <- msg:
			sentCount++
		default:
			gs.logf("[WARN] Subscriber %d channel full, closing slow", s.ID())
			s.closeSlow()
		}
	}
	gs.logf("[PUBLISH] Message sent to %d/%d subscribers in lobby '%s'", sentCount, subscriberCount, lobbyID)
}

// handles incoming chat messages
func (gs *GameServer) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, 8192)
	msgData, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	gs.logf("[CHAT] Received chat message: %s", string(msgData))

	var chatMsg ChatMessage
	if err := json.Unmarshal(msgData, &chatMsg); err != nil {
		gs.logf("[ERROR] Failed to parse chat message: %v", err)
		http.Error(w, "Invalid message format", http.StatusBadRequest)
		return
	}

	if chatMsg.Timestamp == 0 {
		chatMsg.Timestamp = time.Now().Unix()
	}

	chatMsg.Type = "CHAT_MESSAGE"

	gs.logf("[CHAT] User %d sending message to lobby '%s': %s", chatMsg.SenderID, chatMsg.LobbyID, chatMsg.Message)

	// Serialize and publish to the lobby
	msg, _ := json.Marshal(chatMsg)
	gs.publishToLobby(chatMsg.LobbyID, msg)

	w.WriteHeader(http.StatusAccepted)
}

// handles requests to join a specific lobby
func (gs *GameServer) joinLobbyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	var req struct {
		UserID  int    `json:"user_id"`
		LobbyID string `json:"lobby_id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual lobby switching logic
	gs.logf("User %d requested to join lobby %s", req.UserID, req.LobbyID)

	w.WriteHeader(http.StatusAccepted)
}

// handles matchmaking queue join requests
func (gs *GameServer) joinQueueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body := http.MaxBytesReader(w, r.Body, 8192)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		return
	}

	var req struct {
		UserID int `json:"user_id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	gs.queueMutex.Lock()
	gs.matchmakingQueue = append(gs.matchmakingQueue, req.UserID)
	queueSize := len(gs.matchmakingQueue)
	gs.logf("User %d joined queue. Queue size: %d", req.UserID, queueSize)

	// Check if we can make a match (2+ players)
	if queueSize >= 2 {
		player1 := gs.matchmakingQueue[0]
		player2 := gs.matchmakingQueue[1]
		gs.matchmakingQueue = gs.matchmakingQueue[2:]
		gs.queueMutex.Unlock()

		gs.logf("User %d and User %d were matched", player1, player2)

		// Start match in a goroutine
		go gs.startMatch(player1, player2)
	} else {
		gs.queueMutex.Unlock()
	}

	w.WriteHeader(http.StatusAccepted)
}

// startMatch simulates a match between two players
// TODO: Flesh out this game logic. Make it run in goroutine so if the game logic crashes, it doesnt crash server? Or would the function block just end anyways
func (gs *GameServer) startMatch(player1, player2 int) {
	rand.Seed(time.Now().UnixNano())
	winner := player1
	loser := player2
	if rand.Intn(2) == 1 {
		winner = player2
		loser = player1
	}

	gs.logf("Match result: User %d wins against User %d", winner, loser)

	// Send match result
	result := MatchResult{
		Type:     "MATCH_RESULT",
		WinnerID: winner,
		LoserID:  loser,
	}
	msg, _ := json.Marshal(result)
	gs.publish(msg)

	// Store match result in Supabase items table
	// TODO: data engineering for dedicated match table and user results
	// How to make user results account based but be your own?
	go gs.storeMatchResult(winner)

	time.Sleep(6 * time.Second) // TODO: defult time for result showing

	gs.logf("Returning players %d and %d to global lobby", player1, player2)
}

// generateLobbyID generates a unique lobby ID
// TODO: make sure its unique
func generateLobbyID() string {
	return "lobby_" + randomString(8)
}

// TODO: refactor into utils
// randomString generates a random alphanumeric string
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// storeMatchResult stores the match result in the Supabase items table
func (gs *GameServer) storeMatchResult(winnerID int) {
	timestamp := time.Now().Unix()
	title := "match_result" + strconv.FormatInt(timestamp, 10)
	description := strconv.Itoa(winnerID)

	// Use a system user ID for owner_id (test user for now)
	// TODO: remove this hardcoded ID and ensure that system stuff uses the admin acc
	systemUserID := "fb490750-7386-4150-a047-2317d2f51e5b"

	itemData := map[string]interface{}{
		"id":          uuid.New().String(), // Generate UUID for id column
		"title":       title,
		"description": description,
		"owner_id":    systemUserID, // Required for RLS policies
	}

	// Get system client (uses secret key, bypasses RLS)
	client := gs.dbClient.GetSystemClient()

	// REsponse body and row count ignored
	_, _, err := client.From("items").Insert(itemData, false, "", "", "").Execute()
	if err != nil {
		gs.logf("[ERROR] Failed to store match result: %v", err)
		return
	}

	gs.logf("[SUCCESS] Match result stored: title=%s, winner=%d, owner=%s", title, winnerID, systemUserID)
}
