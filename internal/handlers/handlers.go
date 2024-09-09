package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
)

func generateShortID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
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

	event := file.Event{
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		UUID:        uuid.New().String(),
	}

	storage.Mu.Lock()
	storage.Urls = append(storage.Urls, event)
	storage.Mu.Unlock()

	err = storage.SaveURL(&event)
	if err != nil {
		http.Error(w, "Failed to save URLs", http.StatusInternalServerError)
		return
	}

	storage.Mu.Lock()
	storage.UrlStore[shortID] = originalURL
	storage.Mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func HandleJSONPost(w http.ResponseWriter, r *http.Request) {
	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.URL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	shortID := generateShortID()
	shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

	resp := models.Response{
		Result: shortURL,
	}

	event := file.Event{
		OriginalURL: req.URL,
		ShortURL:    shortURL,
		UUID:        uuid.New().String(),
	}

	storage.Mu.Lock()
	storage.Urls = append(storage.Urls, event)
	storage.Mu.Unlock()

	errSave := storage.SaveURL(&event)
	if errSave != nil {
		http.Error(w, "Failed to save URLs", http.StatusInternalServerError)
		return
	}

	storage.Mu.Lock()
	storage.UrlStore[shortID] = req.URL
	storage.Mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}

}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "shortURL")
	if id == "" {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	storage.Mu.Lock()
	originalURL, ok := storage.UrlStore[id]
	storage.Mu.Unlock()

	if !ok {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	if err := storage.DB.Ping(); err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
