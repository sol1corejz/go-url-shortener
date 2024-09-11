package storage

import (
	"database/sql"
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/models"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	query := `
        CREATE TABLE IF NOT EXISTS short_urls (
            id SERIAL PRIMARY KEY,
            short_url VARCHAR(255) UNIQUE NOT NULL,
            original_url TEXT NOT NULL
        );
    `
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Save(data models.URLData) error {
	query := `INSERT INTO short_urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING`
	_, err := p.db.Exec(query, data.ShortURL, data.OriginalURL)
	return err
}

func (p *PostgresStorage) Get(shortID string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM short_urls WHERE short_url = $1`
	err := p.db.QueryRow(query, shortID).Scan(&originalURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", err
	}
	return originalURL, nil
}

func (p *PostgresStorage) Ping() error {
	return p.db.Ping()
}
