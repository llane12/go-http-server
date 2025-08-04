package main

import (
	"chirpy/internal/auth"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// Fictional 3rd-party payment processor system called Polka

func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	// Authorization
	apiKey, err := auth.GetAPIKey(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header", err)
		return
	}

	if apiKey != cfg.polkaKey {
		respondWithError(w, http.StatusUnauthorized, "Invalid API key", err)
		return
	}

	// Decode request
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil || len(params.Event) == 0 {
		respondWithError(w, http.StatusInternalServerError, "Error decoding parameters", err)
		return
	}

	// Check Event type
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Update user record
	dbUser, err := cfg.dbQueries.GetUser(req.Context(), params.Data.UserID)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !dbUser.IsChirpyRed {
		dbUser, err = cfg.dbQueries.UpdateUserToChirpyRed(req.Context(), dbUser.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
