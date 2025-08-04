package main

import (
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerAddChirp(w http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	// Authorization
	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header", err)
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error validating token", err)
		return
	}

	// Decode request
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil || len(params.Body) == 0 {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	// Validation
	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}
	cleanedBody := getCleanedBody(params.Body)

	// Write to database
	dbChirp, err := cfg.dbQueries.CreateChirp(req.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating chirp", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, mapChirp(dbChirp))
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {
	sortString := req.URL.Query().Get("sort")
	authorIDString := req.URL.Query().Get("author_id")

	desc := false
	if len(sortString) > 0 {
		if sortString != "asc" && sortString != "desc" {
			respondWithError(w, http.StatusBadRequest, "Invalid sort parameter value", nil)
			return
		}

		desc = sortString == "asc"
	}

	var dbChirps []database.Chirp

	if len(authorIDString) > 0 {
		authorID, err := uuid.Parse(authorIDString)
		if err != nil || len(authorID) == 0 {
			respondWithError(w, http.StatusBadRequest, "Invalid author_id", err)
			return
		}

		dbChirps, err = cfg.dbQueries.GetChirpsByUser(req.Context(), authorID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error getting chirps", err)
			return
		}
	} else {
		dbChirps2, err := cfg.dbQueries.GetChirps(req.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error getting chirps", err)
			return
		}
		dbChirps = dbChirps2
	}

	slices.SortFunc(dbChirps, func(a, b database.Chirp) int {
		if desc {
			return a.CreatedAt.Compare(b.CreatedAt)
		} else {
			return b.CreatedAt.Compare(a.CreatedAt)
		}
	})

	chirps := make([]Chirp, 0, len(dbChirps))
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, mapChirp(dbChirp))
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, req *http.Request) {
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil || len(chirpID) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid chirpID", err)
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpID)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting chirp", err)
		return
	}

	respondWithJSON(w, http.StatusOK, mapChirp(dbChirp))
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, req *http.Request) {
	// Authorization
	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header", err)
		return
	}

	userID, err := auth.ValidateJWT(tokenString, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error validating token", err)
		return
	}

	// Get chirp
	chirpID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil || len(chirpID) == 0 {
		respondWithError(w, http.StatusBadRequest, "Invalid chirpID", err)
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirp(req.Context(), chirpID)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting chirp", err)
		return
	} else if dbChirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "", err)
		return
	}

	err = cfg.dbQueries.DeleteChirp(req.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mapChirp(dbChirp database.Chirp) Chirp {
	return Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserId:    dbChirp.UserID,
	}
}

func getCleanedBody(body string) string {
	profanities := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	words := strings.Split(body, " ")
	for i, word := range words {
		if _, ok := profanities[strings.ToLower(word)]; ok {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}
