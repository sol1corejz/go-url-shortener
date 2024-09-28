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
	"github.com/sol1corejz/go-url-shortener/internal/auth"
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

func checkIsAuthorized(r *http.Request) (string, error) {
	cookie, err := r.Cookie("token")
	if err != nil {
		return "", err
	}

	userID := auth.GetUserID(cookie.Value)
	if userID == "" {
		return "", err
	}
	return userID, nil
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		token, err := auth.GenerateToken()
		if err != nil {
			http.Error(w, "Unable to generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HttpOnly: true,
		})

		userID = auth.GetUserID(token)
	} else if err != nil {
		http.Error(w, "Error retrieving cookie", http.StatusBadRequest)
		return
	} else {
		userID = auth.GetUserID(cookie.Value)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
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

	event := models.URLData{
		OriginalURL: originalURL,
		ShortURL:    shortID,
		UUID:        uuid.New().String(),
		UserUUID:    userID,
		DeletedFlag: false,
	}

	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		if err = storage.SaveURL(&event); err != nil {
			if errors.Is(err, storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(fmt.Sprintf("%s/%s", config.FlagBaseURL, storage.ExistingShortURL)))

				storage.ExistingShortURL = ""
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

	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		token, err := auth.GenerateToken()
		if err != nil {
			http.Error(w, "Unable to generate token", http.StatusInternalServerError)
			return
		}

		fmt.Println(token)

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HttpOnly: true,
		})

		userID = auth.GetUserID(token)
	} else if err != nil {
		http.Error(w, "Error retrieving cookie", http.StatusBadRequest)
		return
	} else {
		userID = auth.GetUserID(cookie.Value)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

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
		UserUUID:    userID,
		DeletedFlag: false,
	}

	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		if err := storage.SaveURL(&event); err != nil {

			if errors.Is(err, storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)

				resp := models.Response{
					Result: fmt.Sprintf("%s/%s", config.FlagBaseURL, storage.ExistingShortURL),
				}
				json.NewEncoder(w).Encode(resp)

				storage.ExistingShortURL = ""
				return
			}

			http.Error(w, "Failed to save URL", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

func HandleBatchPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		token, err := auth.GenerateToken()
		if err != nil {
			http.Error(w, "Unable to generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HttpOnly: true,
		})

		userID = auth.GetUserID(token)
	} else if err != nil {
		http.Error(w, "Error retrieving cookie", http.StatusBadRequest)
		return
	} else {
		userID = auth.GetUserID(cookie.Value)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

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
					UserUUID:    userID,
					DeletedFlag: false,
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

	originalURL, deleted, ok := storage.GetOriginalURL(id)
	if !ok {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	if deleted {
		http.Error(w, "URL deleted", http.StatusGone)
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

func HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {

	userID, err := checkIsAuthorized(r)

	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := storage.GetURLsByUser(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve URLs", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(urls); err != nil {
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := checkIsAuthorized(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var ids []string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if err = json.Unmarshal(body, &ids); err != nil {
		logger.Log.Info("cannot decode batch request JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(ids) == 0 {
		http.Error(w, "Batch cannot be empty", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)

	go processDeleteBatch(ids, userID)
}

func processDeleteBatch(ids []string, userID string) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	inputCh := generator(doneCh, ids)
	channels := fanOut(doneCh, inputCh, userID)
	errorCh := fanIn(doneCh, channels...)

	for err := range errorCh {
		if err != nil {
			logger.Log.Error("Failed to delete URL", zap.Error(err))
		}
	}
}

func generator(doneCh chan struct{}, ids []string) chan string {
	inputCh := make(chan string)
	go func() {
		defer close(inputCh)
		for _, id := range ids {
			select {
			case <-doneCh:
				return
			case inputCh <- id:
			}
		}
	}()
	return inputCh
}

func deleteUrl(doneCh chan struct{}, inputCh chan string, userID string) chan error {
	resultCh := make(chan error)
	go func() {
		defer close(resultCh)
		for id := range inputCh {
			err := storage.BatchUpdateDeleteFlag(id, userID)
			select {
			case <-doneCh:
				return
			case resultCh <- err:
			}
		}
	}()
	return resultCh
}

func fanOut(doneCh chan struct{}, inputCh chan string, userID string) []chan error {
	numWorkers := 5
	channels := make([]chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		channels[i] = deleteUrl(doneCh, inputCh, userID)
	}
	return channels
}

func fanIn(doneCh chan struct{}, resultChs ...chan error) chan error {
	finalCh := make(chan error)
	var wg sync.WaitGroup

	for _, ch := range resultChs {
		wg.Add(1)

		chClosure := ch

		go func() {
			defer wg.Done()

			for err := range chClosure {
				select {
				case <-doneCh:
					return
				case finalCh <- err:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()

	return finalCh
}
