// Package file предоставляет функции для записи и чтения данных в файле с использованием JSON.
// Он включает структуры и методы для создания производителей и потребителей,
// которые могут записывать и читать события, связанные с URL-данными.
package file

import (
	"encoding/json"
	"os"

	"github.com/sol1corejz/go-url-shortener/internal/models"
)

// Producer представляет собой производителя, который записывает события в файл в формате JSON.
type Producer struct {
	File    *os.File      // Файл, в который будут записываться данные.
	encoder *json.Encoder // JSON-энкодер для сериализации данных.
}

// Consumer представляет собой потребителя, который читает события из файла в формате JSON.
type Consumer struct {
	File    *os.File      // Файл, из которого будут читаться данные.
	decoder *json.Decoder // JSON-декодер для десериализации данных.
}

// NewProducer создает нового производителя, который будет записывать данные в указанный файл.
// Принимает имя файла, в который будет производиться запись. Возвращает объект Producer и ошибку, если таковая возникла.
func NewProducer(fileName string) (*Producer, error) {
	// Открывает файл для записи, создавая его при необходимости.
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Возвращает новый объект Producer с файлом и JSON-энкодером.
	return &Producer{
		File:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// WriteEvent записывает событие в файл в формате JSON.
// Принимает объект event типа *models.URLData и сериализует его в JSON.
func (p *Producer) WriteEvent(event *models.URLData) error {
	return p.encoder.Encode(&event)
}

// NewConsumer создает нового потребителя, который будет читать данные из указанного файла.
// Принимает имя файла, из которого будет происходить чтение. Возвращает объект Consumer и ошибку, если таковая возникла.
func NewConsumer(fileName string) (*Consumer, error) {
	// Открывает файл для чтения, создавая его при необходимости.
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	// Возвращает новый объект Consumer с файлом и JSON-декодером.
	return &Consumer{
		File:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

// ReadEvent читает следующее событие из файла в формате JSON и десериализует его.
// Возвращает объект event типа *models.URLData и ошибку, если чтение или десериализация не удались.
func (c *Consumer) ReadEvent() (*models.URLData, error) {
	event := &models.URLData{}
	if err := c.decoder.Decode(&event); err != nil {
		return nil, err
	}

	return event, nil
}
