package storage

import (
	"encoding/json"
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"os"
	"sync"
)

type FileStorage struct {
	filename string
	mu       sync.Mutex
	data     map[string]string
}

func NewFileStorage(filename string) (*FileStorage, error) {
	fs := &FileStorage{
		filename: filename,
		data:     make(map[string]string),
	}

	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&fs.data)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return fs, nil
}

func (fs *FileStorage) Save(data models.URLData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.data[data.ShortURL] = data.OriginalURL

	file, err := os.OpenFile(fs.filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(fs.data)
	if err != nil {
		return err
	}

	return nil
}

func (fs *FileStorage) Get(shortID string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	originalURL, ok := fs.data[shortID]
	if !ok {
		return "", fmt.Errorf("URL not found")
	}
	return originalURL, nil
}

func (fs *FileStorage) Ping() error {
	return nil
}
