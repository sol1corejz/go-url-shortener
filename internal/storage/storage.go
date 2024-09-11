package storage

import "github.com/sol1corejz/go-url-shortener/internal/models"

type Storage interface {
	Save(data models.URLData) error
	Get(shortID string) (string, error)
	Ping() error
}
