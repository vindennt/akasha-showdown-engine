package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	// Correcting implementation plan to use standard library or Chi?
	// Wait, I said I would use standard library in the plan.
	// "github.com/go-chi/chi/v5"
	// I should use standard library context to get user.
	// path identifiers like {id} in standard library 1.22+ are supported but usage might differ.
	// The user's go.mod says go 1.24.4, so I can use new routing features.

	"github.com/supabase-community/postgrest-go"
	"github.com/vindennt/akasha-showdown-engine/internal/auth"
	"github.com/vindennt/akasha-showdown-engine/internal/db"
	"github.com/vindennt/akasha-showdown-engine/internal/models"
)

type ItemHandler struct {
	dbClient *db.Client
}

func NewItemHandler(dbClient *db.Client) *ItemHandler {
	return &ItemHandler{
		dbClient: dbClient,
	}
}

// Helper to get token from context and apply to client
func (h *ItemHandler) getClient(r *http.Request) *postgrest.Client {
    // Create new client per request to avoid shared state

    authHeader := r.Header.Get("Authorization")
    token := ""
    if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
        token = authHeader[7:]
    }
    
	return h.dbClient.GetUserClient(token)
}

func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
    user, ok := r.Context().Value(auth.UserContextKey).(models.User)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

	var req models.ItemCreate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

    // Prepare item for insertion
    // The prompt says "Set owner_id to the current user's ID".
    // We can insert a map or a struct. We need a struct that includes OwnerID but otherwise matches ItemCreate.
    // Or we can use the keys directly.
    
    // Let's create a map to be safe with partial updates/inserts or create a specific struct.
    // Use a map for flexibility.
    itemData := map[string]interface{}{
        "title": req.Title,
        "owner_id": user.ID,
    }
    if req.Description != nil {
        itemData["description"] = *req.Description
    }

	resp, _, err := h.getClient(r).From("items").Insert(itemData, false, "", "", "").Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    w.Header().Set("Content-Type", "application/json")
    w.Write(resp)
}

func (h *ItemHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+ routing
    if id == "" {
        http.Error(w, "Missing ID", http.StatusBadRequest)
        return
    }

	resp, _, err := h.getClient(r).From("items").Select("*", "", false).Eq("id", id).Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
    
    // PostgREST returns a list. We need to check if it's empty.
    // If we want a single object, postgrest-go returns raw JSON bytes.
    // We can assume if resp is "[]" it's not found? Or just return the list?
    // The prompt says "Response: Item or null (or 404)".
    // If list is empty valid JSON is "[]".
    
    // Let's inspect the response.
    if string(resp) == "[]" {
        http.Error(w, "Not found", http.StatusNotFound)
        return 
    }
    
    // Retrieve the first item
    var items []models.Item
    if err := json.Unmarshal(resp, &items); err != nil {
         http.Error(w, err.Error(), http.StatusInternalServerError)
         return
    }
    
    if len(items) == 0 {
         http.Error(w, "Not found", http.StatusNotFound)
         return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items[0])
}

func (h *ItemHandler) ListItems(w http.ResponseWriter, r *http.Request) {
    skip := 0
    limit := 100
    
    if s := r.URL.Query().Get("skip"); s != "" {
        if v, err := strconv.Atoi(s); err == nil {
            skip = v
        }
    }
    if l := r.URL.Query().Get("limit"); l != "" {
        if v, err := strconv.Atoi(l); err == nil {
            limit = v
        }
    }

    // Range is start-end (inclusive)
    from := skip
    to := skip + limit - 1

	resp, _, err := h.getClient(r).From("items").Select("*", "", false).Range(from, to, "").Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    w.Header().Set("Content-Type", "application/json")
    w.Write(resp)
}

func (h *ItemHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if id == "" {
        http.Error(w, "Missing ID", http.StatusBadRequest)
        return
    }

	var req models.ItemUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, _, err := h.getClient(r).From("items").Update(req, "", "").Eq("id", id).Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    // Check if updated happened
    if string(resp) == "[]" {
         http.Error(w, "Not found or unauthorized", http.StatusNotFound)
         return       
    }

    w.Header().Set("Content-Type", "application/json")
    // Return the updated item (first in list)
    // PostgREST returns the updated rows
    var items []models.Item
    if err := json.Unmarshal(resp, &items); err == nil && len(items) > 0 {
         json.NewEncoder(w).Encode(items[0])
         return
    }
    w.Write(resp) 
}

func (h *ItemHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    if id == "" {
        http.Error(w, "Missing ID", http.StatusBadRequest)
        return
    }

	resp, _, err := h.getClient(r).From("items").Delete("", "").Eq("id", id).Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
    
    if string(resp) == "[]" {
         http.Error(w, "Not found or unauthorized", http.StatusNotFound)
         return       
    }

    w.Header().Set("Content-Type", "application/json")
    // Return the deleted item
    var items []models.Item
    if err := json.Unmarshal(resp, &items); err == nil && len(items) > 0 {
         json.NewEncoder(w).Encode(items[0])
         return
    }
    w.Write(resp)
}
