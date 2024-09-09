package storage

import (
	"database/sql"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"sync"
)

var (
	UrlStore = make(map[string]string)
	Urls     []file.Event
	Mu       sync.Mutex
	DB       *sql.DB
)

func LoadURLs() error {
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
		Urls = append(Urls, *event)
	}

	return nil
}

func SaveURL(event *file.Event) error {
	producer, err := file.NewProducer(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer producer.File.Close()

	return producer.WriteEvent(event)
}
