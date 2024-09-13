package main

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/cmd/gzip"
	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func main() {
	config.ParseFlags()

	var err error

	if config.DatabaseDSN != "" {
		err = storage.NewPostgresStorage()
		if err != nil {
			logger.Log.Fatal("Failed to connect to DB: ", zap.String("DB", config.DatabaseDSN))
		}
	} else if config.FileStoragePath != "" {
		err = storage.LoadURLs()
		if err != nil {
			logger.Log.Fatal("Failed to load URLs from file: ", zap.String("file", config.FileStoragePath))
		}
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

	r.Post("/", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandlePost)))
	r.Get("/{shortURL}", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleGet)))
	r.Post("/api/shorten", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleJSONPost)))
	r.Get("/ping", logger.RequestLogger(handlers.HandlePing))

	return http.ListenAndServe(config.FlagRunAddr, r)
}
