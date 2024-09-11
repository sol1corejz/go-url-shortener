package storage

import (
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"sync"
)

type FileStorage struct {
	filename string
	mu       sync.Mutex
	data     map[string]models.URLData
	producer *file.Producer
	consumer *file.Consumer
}

func NewFileStorage(filename string) (*FileStorage, error) {
	fs := &FileStorage{
		filename: filename,
		data:     make(map[string]models.URLData),
	}

	producer, err := file.NewProducer(filename)
	if err != nil {
		return nil, err
	}
	fs.producer = producer

	consumer, err := file.NewConsumer(filename)
	if err != nil {
		return nil, err
	}
	fs.consumer = consumer

	for {
		event, err := fs.consumer.ReadEvent()
		if err != nil {
			break
		}
		fs.data[event.ShortURL] = models.URLData{
			UUID:        event.UUID,
			ShortURL:    event.ShortURL,
			OriginalURL: event.OriginalURL,
		}
	}

	return fs, nil
}

func (fs *FileStorage) saveToFile(data models.URLData) error {
	event := &file.Event{
		UUID:        data.UUID,
		ShortURL:    data.ShortURL,
		OriginalURL: data.OriginalURL,
	}
	return fs.producer.WriteEvent(event)
}

func (fs *FileStorage) Save(data models.URLData) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.data[data.ShortURL] = data
	return fs.saveToFile(data)
}

func (fs *FileStorage) Get(shortID string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if data, exists := fs.data[shortID]; exists {
		return data.OriginalURL, nil
	}

	return "", fmt.Errorf("URL not found")
}

func (fs *FileStorage) Ping() error {
	return nil
}
