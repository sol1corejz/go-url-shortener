package storage

import (
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"strings"
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

	shortID := data.ShortURL[strings.LastIndex(data.ShortURL, "/")+1:]

	ms.data[shortID] = data.OriginalURL
	return nil
}

func (ms *MemoryStorage) Get(shortURL string) (string, error) {
	originalURL, found := ms.data[shortURL]
	if !found {
		return "", fmt.Errorf("not found")
	}

	return originalURL, nil
}

func (ms *MemoryStorage) Ping() error {
	return nil
}
