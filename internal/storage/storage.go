package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"go.uber.org/zap"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	URLStore         = make(map[string]string)
	Mu               sync.Mutex
	DB               *sql.DB
	ExistingShortURL string
	ErrAlreadyExists = errors.New("ссылка уже сокращена")
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
				original_url TEXT NOT NULL UNIQUE,
			    user_id TEXT NOT NULL,
			    is_deleted BOOLEAN NOT NULL
			)
		`)
		if err != nil {
			logger.Log.Error("Error creating table", zap.Error(err))
			return
		}

		err = loadURLsFromDB()
		if err != nil {
			logger.Log.Error("Error loading URLs from DB", zap.Error(err))
			return
		}

	} else if config.FileStoragePath != "" {
		err := loadURLsFromFile()
		if err != nil {
			logger.Log.Error("Error loading URLs from file", zap.Error(err))
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

		err := DB.QueryRow(`
			SELECT short_url FROM short_urls WHERE original_url=$1
		`, event.OriginalURL).Scan(&ExistingShortURL)

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
			} else {
				return err
			}
		}

		if ExistingShortURL != "" {
			event.ShortURL = ExistingShortURL
			return ErrAlreadyExists
		}

		_, err = DB.Exec(`
			INSERT INTO short_urls (short_url, original_url, user_id, is_deleted) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (original_url)
			DO UPDATE SET short_url = short_urls.short_url
		`, event.ShortURL, event.OriginalURL, event.UserUUID, event.DeletedFlag)

		if err != nil {
			return err
		}

		return nil
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

func SaveSingleURL(event *models.URLData) error {
	if DB != nil {
		_, err := DB.Exec("INSERT INTO short_urls (short_url, original_url, user_id, is_deleted) VALUES ($1, $2, $3, $4) ON CONFLICT (original_url) DO NOTHING;", event.ShortURL, event.OriginalURL, event.UserUUID, event.DeletedFlag)
		fmt.Println(err)
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
		if err := SaveSingleURL(&event); err != nil {
			return errors.New("failed to save batch URLs")
		}
	}

	return nil
}

func GetOriginalURL(shortID string) (string, bool, bool) {
	if DB != nil {
		var originalURL string
		var deleted bool
		err := DB.QueryRow("SELECT original_url, is_deleted, FROM short_urls WHERE short_url = $1", shortID).Scan(&originalURL, &deleted)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", false, false
			}
			return "", false, false
		}
		return originalURL, deleted, true
	}

	Mu.Lock()
	defer Mu.Unlock()
	originalURL, ok := URLStore[shortID]
	return originalURL, false, ok
}

func GetURLsByUser(userID string) ([]models.URLData, error) {
	if DB != nil {
		rows, err := DB.Query("SELECT short_url, original_url FROM short_urls WHERE user_id = $1", userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var urls []models.URLData
		for rows.Next() {
			var shortURL, originalURL string
			if err := rows.Scan(&shortURL, &originalURL); err != nil {
				return nil, err
			}
			urls = append(urls, models.URLData{ShortURL: config.FlagBaseURL + "/" + shortURL, OriginalURL: originalURL})
		}
		return urls, rows.Err()
	}
	return nil, nil
}

func BatchUpdateDeleteFlag(urlID string, userID string) error {
	query := `UPDATE short_urls SET is_deleted = TRUE WHERE short_url = $1 AND user_id = $2`
	_, err := DB.Exec(query, urlID, userID)
	return err
}
