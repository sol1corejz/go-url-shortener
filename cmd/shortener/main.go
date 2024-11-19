package main

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
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

	r.Mount("/debug/pprof", http.StripPrefix("/debug/pprof", http.HandlerFunc(pprof.Index)))
	r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Index))

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
