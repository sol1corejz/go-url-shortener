package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/cmd/gzip"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	urlStore = make(map[string]string)
	urls     []file.Event
	mu       sync.Mutex
	db       *sql.DB
)

func loadURLs() error {
	consumer, err := file.NewConsumer(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer consumer.File.Close()

	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			break
		}
		urls = append(urls, *event)
	}

	return nil
}

func saveURL(event *file.Event) error {
	producer, err := file.NewProducer(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer producer.File.Close()

	return producer.WriteEvent(event)
}

func generateShortID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
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

	event := file.Event{
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		UUID:        uuid.New().String(),
	}

	mu.Lock()
	urls = append(urls, event)
	mu.Unlock()

	err = saveURL(&event)
	if err != nil {
		http.Error(w, "Failed to save URLs", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	urlStore[shortID] = originalURL
	mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func handleJSONPost(w http.ResponseWriter, r *http.Request) {
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

	event := file.Event{
		OriginalURL: req.URL,
		ShortURL:    shortURL,
		UUID:        uuid.New().String(),
	}

	mu.Lock()
	urls = append(urls, event)
	mu.Unlock()

	errSave := saveURL(&event)
	if errSave != nil {
		http.Error(w, "Failed to save URLs", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	urlStore[shortID] = req.URL
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}

}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "shortURL")
	if id == "" {
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	mu.Lock()
	originalURL, ok := urlStore[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "URL not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func main() {
	config.ParseFlags()

	var err error
	db, err = sql.Open("pgx", config.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = loadURLs()
	if err != nil {
		logger.Log.Fatal("Failed to load URLs from file: ", zap.String("file", config.DefaultFilePath))
	}

	if err := run(); err != nil {
		panic(err)
	}
}

func gzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			cw := gzip.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := gzip.NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}

func run() error {
	if err := logger.Initialize(config.FlagLogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", config.FlagRunAddr))

	r := chi.NewRouter()

	r.Post("/", logger.RequestLogger(gzipMiddleware(handlePost)))
	r.Get("/{shortURL}", logger.RequestLogger(gzipMiddleware(handleGet)))
	r.Post("/api/shorten", logger.RequestLogger(gzipMiddleware(handleJSONPost)))
	r.Get("/ping", logger.RequestLogger(handlePing))

	return http.ListenAndServe(config.FlagRunAddr, r)
}
