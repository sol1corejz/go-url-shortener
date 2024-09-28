package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"time"
)

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
		if existURL, err := storage.SaveURL(&event); err != nil {

			if errors.Is(err, storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)

				resp := models.Response{
					Result: fmt.Sprintf("%s/%s", config.FlagBaseURL, existURL),
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
