package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// HealthzHandler handles the /healthz readiness check
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) adminMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	count := cfg.fileserverHits.Load() // atomic load

	html := fmt.Sprintf(`
		<html>
		  <body>
		    <h1>Welcome, Chirpy Admin</h1>
		    <p>Chirpy has been visited %d times!</p>
		  </body>
		</html>
	`, count)

	w.Write([]byte(html))
}

// POST /admin/reset
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method not allowed", nil)
		return
	}

	if cfg.PLATFORM != "dev" {
		respondWithError(w, http.StatusForbidden, "Forbidden: reset allowed only in dev environment", nil)
		return
	}

	err := cfg.DB.DeleteAllUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete users", err)
		return
	}

	cfg.fileserverHits.Store(0)
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Counter reset"})
}

func handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	params := validateChirpRequest{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	// List of banned words (lowercase)
	bannedWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}

	words := strings.Split(params.Body, " ")
	for i, word := range words {
		if _, banned := bannedWords[strings.ToLower(word)]; banned {
			words[i] = "****"
		}
	}
	cleaned := strings.Join(words, " ")

	respondWithJSON(w, http.StatusOK, validateChirpResponse{
		CleanedBody: cleaned,
	})
}

// POST /api/users
func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	userFromDB, err := cfg.DB.CreateUser(r.Context(), req.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create user", err)
		return
	}

	// Map database.User to your User struct to control JSON response keys
	user := User{
		ID:        userFromDB.ID,
		CreatedAt: userFromDB.CreatedAt,
		UpdatedAt: userFromDB.UpdatedAt,
		Email:     userFromDB.Email,
	}

	respondWithJSON(w, http.StatusCreated, user)
}
