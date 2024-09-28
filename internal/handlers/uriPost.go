package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"io"
	"net/http"
	"strings"
	"time"
)

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
		if existURL, err := storage.SaveURL(&event); err != nil {
			if errors.Is(err, storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(fmt.Sprintf("%s/%s", config.FlagBaseURL, existURL)))

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
