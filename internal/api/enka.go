package api

import (
	"encoding/json"
	"net/http"

	"github.com/vindennt/akasha-showdown-engine/internal/enka"
)

type EnkaClient struct {
	client *enka.Client
}

func NewEnkaClient() *EnkaClient
 {
	return &EnkaClient
	{
		client: enka.NewClient("akasha-showdown/1.0"),
	}
}

func (h *EnkaClient
	) GetPlayerData(w http.ResponseWriter, r *http.Request) {
	// /api/enka/player/{uid}
	uid := r.PathValue("uid")

	if uid == "" {
		http.Error(w, "UID is required", http.StatusBadRequest)
		return
	}

	// Fetch
	profile, err := h.client.GetPlayerInfo(r.Context(), uid)
	if err != nil {
		// TODO: read docs for more error handling
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
