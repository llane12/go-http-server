package main

import (
	"chirpy/internal/auth"
	"database/sql"
	"net/http"
	"time"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, req *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Authorization header", err)
		return
	}

	refreshToken, err := cfg.dbQueries.GetRefreshToken(req.Context(), tokenString)
	if err == sql.ErrNoRows ||
		refreshToken.RevokedAt.Valid || // RevokedAt is not null, so the refresh token has been revoked
		refreshToken.ExpiresAt.Before(time.Now().UTC()) { // Refresh token has expired
		respondWithError(w, http.StatusUnauthorized, "", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting refresh token", err)
		return
	}

	user, err := cfg.dbQueries.GetUserFromRefreshToken(req.Context(), tokenString)
	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusUnauthorized, "", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting user record", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.tokenSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error generating JWT", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, req *http.Request) {
	tokenString, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Authorization header", err)
		return
	}

	err = cfg.dbQueries.RevokeRefreshToken(req.Context(), tokenString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh token", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
