package storage

import (
	"context"
	"database/sql"
	"errors"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"go.uber.org/zap"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	URLStore = make(map[string]string)
	Mu       sync.Mutex
	DB       *sql.DB
)

func InitializeStorage(ctx context.Context) {
	if config.DatabaseDSN != "" {

		db, err := sql.Open("pgx", config.DatabaseDSN)
		if err != nil {
			logger.Log.Fatal("Error opening database connection", zap.Error(err))
			return
		}

		DB = db

		_, err = DB.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS short_urls (
				id SERIAL PRIMARY KEY,
				short_url TEXT NOT NULL UNIQUE,
				original_url TEXT NOT NULL
			)
		`)
		if err != nil {
			logger.Log.Info("Error creating table", zap.Error(err))
			return
		}

		err = loadURLsFromDB()
		if err != nil {
			logger.Log.Info("Error loading URLs from DB", zap.Error(err))
			return
		}

	} else if config.FileStoragePath != "" {
		err := loadURLsFromFile()
		if err != nil {
			logger.Log.Info("Error loading URLs from file", zap.Error(err))
			return
		}
	}
}

func loadURLsFromDB() error {
	rows, err := DB.Query("SELECT short_url, original_url FROM short_urls")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var shortURL, originalURL string
		if err = rows.Scan(&shortURL, &originalURL); err != nil {
			return err
		}
		URLStore[shortURL] = originalURL
	}
	return rows.Err()
}

func loadURLsFromFile() error {
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
		URLStore[event.ShortURL] = event.OriginalURL
	}
	return nil
}

func SaveURL(event *models.URLData) error {
	if DB != nil {
		_, err := DB.Exec("INSERT INTO short_urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING", event.ShortURL, event.OriginalURL)
		return err
	} else if config.FileStoragePath != "" {
		producer, err := file.NewProducer(config.FileStoragePath)
		if err != nil {
			return err
		}
		defer producer.File.Close()

		Mu.Lock()
		URLStore[event.ShortURL] = event.OriginalURL
		Mu.Unlock()

		return producer.WriteEvent(event)
	}

	Mu.Lock()
	URLStore[event.ShortURL] = event.OriginalURL
	Mu.Unlock()
	return nil
}

func SaveBatchURL(events []models.URLData) error {
	Mu.Lock()
	defer Mu.Unlock()

	for _, event := range events {
		if err := SaveURL(&event); err != nil {
			return errors.New("failed to save batch URLs")
		}
	}

	return nil
}

func GetOriginalURL(shortID string) (string, bool) {
	if DB != nil {
		var originalURL string
		err := DB.QueryRow("SELECT original_url FROM short_urls WHERE short_url = $1", shortID).Scan(&originalURL)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", false
			}
			return "", false
		}
		return originalURL, true
	}

	Mu.Lock()
	defer Mu.Unlock()
	originalURL, ok := URLStore[shortID]
	return originalURL, ok
}
