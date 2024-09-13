package storage

import (
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
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
		URLs = append(URLs, *event)
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
