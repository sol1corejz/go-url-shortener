package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
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
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

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

	event := models.URLData{
		OriginalURL: originalURL,
		ShortURL:    shortID,
		UUID:        uuid.New().String(),
	}

	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		if err = storage.SaveURL(&event); err != nil {

			if errors.As(err, &storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				resp := models.Response{
					Result: fmt.Sprintf("%s/%s", config.FlagBaseURL, storage.ExistingShortURL),
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			http.Error(w, "Failed to save URL", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func HandleJSONPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

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

	event := models.URLData{
		OriginalURL: req.URL,
		ShortURL:    shortID,
		UUID:        uuid.New().String(),
	}

	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		if err := storage.SaveURL(&event); err != nil {

			if errors.As(err, &storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				resp := models.Response{
					Result: fmt.Sprintf("%s/%s", config.FlagBaseURL, storage.ExistingShortURL),
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			http.Error(w, "Failed to save URL", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

func HandleBatchPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req []models.BatchRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if err = json.Unmarshal(body, &req); err != nil {
		logger.Log.Info("cannot decode batch request JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "Batch cannot be empty", http.StatusBadRequest)
		return
	}

	var res []models.BatchResponse
	var wg sync.WaitGroup
	var mu sync.Mutex
	events := make([]models.URLData, len(req))

	for i, item := range req {
		wg.Add(1)

		go func(i int, item models.BatchRequest) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
				shortID := generateShortID()
				shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

				event := models.URLData{
					OriginalURL: item.OriginalURL,
					ShortURL:    shortID,
					UUID:        item.CorrelationID,
				}

				mu.Lock()
				events[i] = event
				res = append(res, models.BatchResponse{
					CorrelationID: item.CorrelationID,
					ShortURL:      shortURL,
				})
				mu.Unlock()
			}
		}(i, item)
	}

	wg.Wait()

	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		if err := storage.SaveBatchURL(events); err != nil {
			http.Error(w, "Failed to save URLs", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

}

func HandleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "shortURL")
	if id == "" {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	originalURL, ok := storage.GetOriginalURL(id)
	if !ok {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(originalURL))
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	if err := storage.DB.Ping(); err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
