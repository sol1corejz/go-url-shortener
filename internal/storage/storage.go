package storage

import (
	"database/sql"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"sync"
)

type Storage interface {
	Save(data models.URLData) error
	Get(shortID string) (string, error)
	Ping() error
}

var (
	URLStore = make(map[string]string)
	URLs     []file.Event
	Mu       sync.Mutex
	DB       *sql.DB
)
