package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/models"
)

func BenchmarkHandlePost(b *testing.B) {
	requestBody := []byte("https://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "text/plain")

		w := httptest.NewRecorder()
		handlers.HandlePost(w, req)

		if w.Code != http.StatusCreated {
			b.Errorf("unexpected status code: got %d, want %d", w.Code, http.StatusCreated)
		}
	}
}

func BenchmarkHandleJSONPost(b *testing.B) {

	requestBody := []byte(`{"url": "https://example.com"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/shorten", io.NopCloser(bytes.NewBuffer(requestBody)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		response := w.Result()
		defer response.Body.Close()

		handlers.HandleJSONPost(w, req)

		if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusConflict {
			b.Errorf("Unexpected status code: %d", response.StatusCode)
		}

	}
}

func BenchmarkHandleBatchPost(b *testing.B) {
	batchRequest := []models.BatchRequest{
		{OriginalURL: "http://example.com/1", CorrelationID: "cor1"},
		{OriginalURL: "http://example.com/2", CorrelationID: "cor2"},
		{OriginalURL: "http://example.com/3", CorrelationID: "cor3"},
	}

	requestBody, err := json.Marshal(batchRequest)
	if err != nil {
		b.Fatalf("failed to marshal request body: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		handlers.HandleBatchPost(w, req)

		if w.Code != http.StatusCreated {
			b.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}
