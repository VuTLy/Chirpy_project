package main

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"main.go/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	PLATFORM       string
	jwtSecret      string // Add this line
}

type validateChirpRequest struct {
	Body string `json:"body"`
}

type validateChirpResponse struct {
	CleanedBody string `json:"cleaned_body"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type createUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
