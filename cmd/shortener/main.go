package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	config.ParseFlags()

	ctx := context.Background()

	storage.InitializeStorage(ctx)

	if err := run(); err != nil {
		logger.Log.Fatal("Failed to run server", zap.Error(err))
	}
}

func run() error {
	if err := logger.Initialize(config.FlagLogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", config.FlagRunAddr))

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Post("/", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandlePost)))
		r.Get("/{shortURL}", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleGet)))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleJSONPost)))
		r.Post("/shorten/batch", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleBatchPost)))
		r.Get("/user/urls", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleGetUserURLs)))
		r.Delete("/user/urls", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleDeleteURLs)))
	})

	r.Get("/ping", logger.RequestLogger(handlers.HandlePing))

	return http.ListenAndServe(config.FlagRunAddr, r)
}
