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
	// Генерация случайного байтового массива
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func handleURL(storage *URLStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)

		if len(body) != 0 {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		} else {
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
}

func main() {

	us := &URLStorage{map[string]string{}}

	//mux := http.NewServeMux()
	http.Handle("/", handleURL(us))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
