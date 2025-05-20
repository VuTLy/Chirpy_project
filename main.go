package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"main.go/internal/database"
)

func main() {
	const filepathRoot = "."
	const port = "8080"

	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Connect to PostgreSQL
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Can't connect to database:", err)
	}

	// Create SQLC query handler
	dbQueries := database.New(db)

	// 🔐 Load JWT secret from env
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set in environment")
	}

	// Create API config with DB access and JWT secret
	apiCfg := &apiConfig{
		DB:        dbQueries,
		PLATFORM:  os.Getenv("PLATFORM"),
		jwtSecret: jwtSecret, // 🔐 Add this line
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", HealthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.adminMetricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", handlerChirpsValidate)
	mux.HandleFunc("/api/users", apiCfg.createUserHandler)
	mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpByIDHandler)
	mux.HandleFunc("/api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	mux.HandleFunc("PUT /api/users", apiCfg.updateUserHandler)
	mux.HandleFunc("/api/chirps/{chirpID}", apiCfg.deleteChirpHandler)

	// Wrap file server with the metrics increment middleware
	fileServer := http.FileServer(http.Dir(filepathRoot))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s at http://localhost:%s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
