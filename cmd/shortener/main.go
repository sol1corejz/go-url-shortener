package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	urlStore = make(map[string]string)
	mu       sync.Mutex
)

func generateShortID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortID := generateShortID()
	shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

	mu.Lock()
	urlStore[shortID] = originalURL
	mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/")
	if id == "" {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	mu.Lock()
	originalURL, ok := urlStore[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func main() {
	config.ParseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("Running server on", config.FlagRunAddr)

	r := chi.NewRouter()

	r.Post("/", handlePost)
	r.Get("/{shortURL}", handleGet)

	return http.ListenAndServe(config.FlagRunAddr, r)
}
