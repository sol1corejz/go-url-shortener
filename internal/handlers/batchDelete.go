package handlers

import (
	"encoding/json"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sync"
)

func HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := auth.CheckIsAuthorized(r)
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

	inputCh := generatorDeleteBatch(doneCh, ids)
	channels := fanOutDeleteBatch(doneCh, inputCh, userID)
	errorCh := fanInDeleteBatch(doneCh, channels...)

	for err := range errorCh {
		if err != nil {
			logger.Log.Error("Failed to delete URL", zap.Error(err))
		}
	}
}

func deleteURL(doneCh chan struct{}, inputCh chan string, userID string) chan error {
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

func generatorDeleteBatch(doneCh chan struct{}, ids []string) chan string {
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

func fanOutDeleteBatch(doneCh chan struct{}, inputCh chan string, userID string) []chan error {
	numWorkers := 5
	channels := make([]chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		channels[i] = deleteURL(doneCh, inputCh, userID)
	}
	return channels
}

func fanInDeleteBatch(doneCh chan struct{}, resultChs ...chan error) chan error {
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
