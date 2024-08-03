package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
)

type URLStorage struct {
	URL map[string]string
}

func (storage *URLStorage) SetURL(newURL string) string {
	shortURL, err := generateID()
	if err != nil {
		return ""
	}

	storage.URL[shortURL] = newURL
	return shortURL
}

func (storage *URLStorage) GetURL(shortURL string) (string, error) {
	value, ok := storage.URL[shortURL]
	if !ok {
		return "", errors.New(shortURL + " not exist")
	}
	return value, nil
}

func generateID() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func handlePostURL(storage *URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		originalURL := string(body)

		shortID := storage.SetURL(originalURL)
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("http://localhost:8080/" + shortID))
	}
}

func handleGetOriginalURL(storage *URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		shortID := strings.TrimPrefix(r.URL.Path, "/")

		if shortID == "" {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		originalURL, err := storage.GetURL(shortID)

		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func main() {

	us := &URLStorage{map[string]string{}}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlePostURL(us).ServeHTTP(w, r)
		case http.MethodGet:
			handleGetOriginalURL(us).ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
