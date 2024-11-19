package handlers

import (
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
	"io"
	"net/http"
	"sync"
	"time"
)

func HandleBatchPost(w http.ResponseWriter, r *http.Request) {
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

	processBatchPost(req, userID, &res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func processBatchPost(req []models.BatchRequest, userID string, res *[]models.BatchResponse) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	inputCh := generatorBatchPost(doneCh, req, userID)
	channels := fanOutBatchPost(doneCh, inputCh)
	resultCh := fanInBatchPost(doneCh, channels...)

	for result := range resultCh {
		*res = append(*res, result)
	}
}

func postURL(doneCh chan struct{}, inputCh chan models.URLData) chan models.BatchResponse {
	resultCh := make(chan models.BatchResponse)
	go func() {
		defer close(resultCh)
		for event := range inputCh {
			batchResponse := models.BatchResponse{
				CorrelationID: event.CorrelationID,
				ShortURL:      "",
			}
			shortURL, err := storage.SaveURL(&event)
			if err != nil {
				if errors.Is(err, storage.ErrAlreadyExists) {
					batchResponse.ShortURL = fmt.Sprintf("%s/%s", config.FlagBaseURL, shortURL)
				}
			} else {
				batchResponse.ShortURL = fmt.Sprintf("%s/%s", config.FlagBaseURL, event.ShortURL)
			}
			select {
			case <-doneCh:
				return
			case resultCh <- batchResponse:
			}
		}
	}()
	return resultCh
}

func generatorBatchPost(doneCh chan struct{}, data []models.BatchRequest, userID string) chan models.URLData {
	inputCh := make(chan models.URLData)
	go func() {
		defer close(inputCh)
		for _, event := range data {
			ev := models.URLData{
				UUID:          uuid.New().String(),
				ShortURL:      generateShortID(),
				OriginalURL:   event.OriginalURL,
				DeletedFlag:   false,
				UserUUID:      userID,
				CorrelationID: event.CorrelationID,
			}
			select {
			case <-doneCh:
				return
			case inputCh <- ev:
			}
		}
	}()
	return inputCh
}

func fanOutBatchPost(doneCh chan struct{}, inputCh chan models.URLData) []chan models.BatchResponse {
	numWorkers := 5
	channels := make([]chan models.BatchResponse, numWorkers)
	for i := 0; i < numWorkers; i++ {
		channels[i] = postURL(doneCh, inputCh)
	}
	return channels
}

func fanInBatchPost(doneCh chan struct{}, resultChs ...chan models.BatchResponse) chan models.BatchResponse {
	finalCh := make(chan models.BatchResponse)
	var wg sync.WaitGroup

	for _, ch := range resultChs {
		wg.Add(1)

		chClosure := ch

		go func() {
			defer wg.Done()

			for res := range chClosure {
				select {
				case <-doneCh:
					return
				case finalCh <- res:
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