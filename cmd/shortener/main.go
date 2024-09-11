package main

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func main() {
	config.ParseFlags()

	store, err := initStorage()
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}

	h := handlers.NewHandler(store)

	if err := run(h); err != nil {
		panic(err)
	}
}

func initStorage() (storage.Storage, error) {
	var store storage.Storage
	var err error

	if config.DatabaseDSN != "" {
		store, err = storage.NewPostgresStorage(config.DatabaseDSN)
		if err != nil {
			return nil, err
		}
		log.Println("Using PostgreSQL as storage")
	} else if config.FileStoragePath != "" {
		store, err = storage.NewFileStorage(config.FileStoragePath)
		if err != nil {
			return nil, err
		}
		log.Println("Using file storage")
	} else {
		store = storage.NewMemoryStorage()
		log.Println("Using in-memory storage")
	}

	return store, nil
}

func run(h *handlers.Handler) error {
	if err := logger.Initialize(config.FlagLogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", config.FlagRunAddr))

	r := chi.NewRouter()

	r.Post("/", logger.RequestLogger(middlewares.GzipMiddleware(h.HandlePost)))
	r.Get("/{shortURL}", logger.RequestLogger(middlewares.GzipMiddleware(h.HandleGet)))
	r.Post("/api/shorten", logger.RequestLogger(middlewares.GzipMiddleware(h.HandleJSONPost)))
	r.Get("/ping", logger.RequestLogger(h.HandlePing))

	return http.ListenAndServe(config.FlagRunAddr, r)
}
