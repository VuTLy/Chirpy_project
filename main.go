package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do something before calling the next handler
		cfg.fileserverHits.Add(1)
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Get the current value of fileserverHits
	hits := cfg.fileserverHits.Load()

	// Write it to the response with formatted HTML
	html := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, hits)

	w.Write([]byte(html))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Reset fileserverHits to 0
	cfg.fileserverHits.Store(0)
	w.Write([]byte("Counter reset"))
}

// JSON SECTION CH3
type opinionRequest struct {
	Body string `json:"body"`
}

// Prohibited words list
var prohibitedWords = []string{"kerfuffle", "sharbert", "fornax"}

func (cfg *apiConfig) validateChirp(w http.ResponseWriter, r *http.Request) {
	// validating the method of the request to post
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// decode and validate for json format
	var req opinionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	//trimming
	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeJSONError(w, http.StatusBadRequest, "Body cannot be empty")
		return
	}

	//splitting the body to words and validate
	words := strings.Fields(body)
	for i, word := range words {
		//lower case the word
		loweredWord := strings.ToLower(word)

		//comparison process
		if contains(prohibitedWords, loweredWord) && !hasPunctuation(word) {
			words[i] = "****"
		}
	}

	modifiedBody := strings.Join(words, " ")

	//Counting limt
	const maxWord = 140
	if len(body) >= maxWord {
		writeJSONError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	//fully validated
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"cleaned_body": modifiedBody})
}

// helper if word is in prohibited list
func contains(list []string, word string) bool {
	for _, item := range list {
		if item == word {
			return true
		}
	}
	return false
}

// helper to validate if a word contain a punctuation
func hasPunctuation(word string) bool {
	for _, ch := range word {
		if unicode.IsPunct(ch) {
			return true
		}
	}
	return false
}

// helper function to json error responses
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	mux := http.NewServeMux()
	// Register your healthz handler
	mux.HandleFunc("GET /api/healthz", healthzHandler)

	fileServer := http.FileServer(http.Dir(filepathRoot))

	// Create an instance of your apiConfig
	apiCfg := &apiConfig{}
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateChirp)

	// Then wrap your file server handler with the middleware
	handler := http.StripPrefix("/app", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}
