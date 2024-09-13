package db

import (
	"database/sql"
	"fmt"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
)

func NewPostgresStorage() error {
	var err error

	storage.DB, err = sql.Open("pgx", config.DatabaseDSN)
	if err != nil {
		panic(err)
	}

	query := `
        CREATE TABLE IF NOT EXISTS short_urls (
            id SERIAL PRIMARY KEY,
            short_url VARCHAR(255) UNIQUE NOT NULL,
            original_url TEXT NOT NULL
        );
    `
	_, err = storage.DB.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func Save(data models.URLData) error {
	query := `INSERT INTO short_urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING`
	_, err := storage.DB.Exec(query, data.ShortURL, data.OriginalURL)
	return err
}

func Get(shortID string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM short_urls WHERE short_url = $1`
	err := storage.DB.QueryRow(query, shortID).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", err
	}
	return originalURL, nil
}
