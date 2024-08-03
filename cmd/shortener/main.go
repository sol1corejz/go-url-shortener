package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
)

type UrlStorage struct {
	url map[string]string
}

func (storage *UrlStorage) SetUrl(newUrl string) string {
	shortUrl, err := generateID()
	if err != nil {
		return ""
	}

	storage.url[shortUrl] = newUrl
	return shortUrl
}

func (storage *UrlStorage) GetUrl(shortUrl string) (string, error) {
	value, ok := storage.url[shortUrl]
	if !ok {
		return "", errors.New(shortUrl + " not exist")
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

func handleUrl(storage *UrlStorage) http.HandlerFunc {
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
			originalUrl := string(body)

			shortId := storage.SetUrl(originalUrl)
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("http://localhost:8080/" + shortId))
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

			originalUrl, err := storage.GetUrl(shortID)

			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusTemporaryRedirect)
			w.Header().Set("Location", originalUrl)
		}
	}
}

func main() {

	us := &UrlStorage{map[string]string{}}

	//mux := http.NewServeMux()
	http.Handle("/", handleUrl(us))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
