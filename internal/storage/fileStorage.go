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
	data     []models.URLData
}

func NewFileStorage(filename string) (*FileStorage, error) {
	fs := &FileStorage{
		filename: filename,
		data:     make([]models.URLData, 0),
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

func (fs *FileStorage) saveToFile() error {
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

func (fs *FileStorage) Save(data models.URLData) error {
	fs.mu.Lock()
	urlData := models.URLData{
		UUID:        data.UUID,
		ShortURL:    data.ShortURL,
		OriginalURL: data.OriginalURL,
	}
	defer fs.mu.Unlock()

	for i, d := range fs.data {
		if d.ShortURL == urlData.ShortURL {
			fs.data[i] = urlData
			return fs.saveToFile()
		}
	}

	fs.data = append(fs.data, urlData)
	return fs.saveToFile()
}

func (fs *FileStorage) Get(shortID string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, d := range fs.data {
		if d.ShortURL == shortID {
			return d.OriginalURL, nil
		}
	}

	return "", fmt.Errorf("URL not found")
}

func (fs *FileStorage) Ping() error {
	return nil
}
