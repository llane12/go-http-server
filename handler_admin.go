package main

import (
	"fmt"
	"net/http"
)

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, req *http.Request) {
	hits := cfg.fileserverHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	body := fmt.Sprintf(`<html>
<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>
</html>`, hits)
	w.Write([]byte(body))
}

func (cfg *apiConfig) handlerReset(resp http.ResponseWriter, req *http.Request) {
	if cfg.platform != "dev" {
		respondWithError(resp, http.StatusForbidden, "You do not have permission to access this resource", nil)
	}

	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Error deleting users", err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte("Hits reset to 0\n"))
	resp.Write([]byte("Users table cleared"))
}
