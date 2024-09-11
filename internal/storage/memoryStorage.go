package storage

import (
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"sync"
)

type MemoryStorage struct {
	mu   sync.Mutex
	data map[string]string
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]string),
	}
}

func (ms *MemoryStorage) Save(data models.URLData) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.data[data.ShortURL] = data.OriginalURL
	return nil
}

func (s *MemoryStorage) Get(shortURL string) (string, error) {
	originalURL, found := s.data[shortURL]
	if !found {
		return "", fmt.Errorf("not found")
	}

	return originalURL, nil
}

func (ms *MemoryStorage) Ping() error {
	return nil
}
