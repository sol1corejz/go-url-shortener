// Package storage предоставляет функции для работы с хранилищем данных
// для сокращённых URL. В зависимости от конфигурации используется либо база данных,
// либо файловое хранилище для сохранения, получения и управления сокращёнными ссылками.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/file"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// URLStore представляет собой мапу для хранения сокращённых URL в памяти.
// Ключом является сокращённый URL, значением - оригинальный URL.
var URLStore = make(map[string]string)

// Mu — мьютекс для синхронизации доступа к URLStore.
var Mu sync.Mutex

// DB представляет собой подключение к базе данных.
var DB *sql.DB

// ExistingShortURL используется для хранения найденного сокращённого URL в базе данных.
var ExistingShortURL string

// ErrAlreadyExists — ошибка, которая возвращается, если сокращённый URL уже существует.
var ErrAlreadyExists = errors.New("ссылка уже сокращена")

// InitializeStorage инициализирует хранилище данных, подключая либо базу данных,
// либо файловое хранилище в зависимости от конфигурации.
// В случае использования базы данных создаёт таблицу для хранения сокращённых URL,
// а затем загружает существующие данные из базы данных или файла.
func InitializeStorage(ctx context.Context) {
	if config.DatabaseDSN != "" {

		// Подключение к базе данных.
		db, err := sql.Open("pgx", config.DatabaseDSN)
		if err != nil {
			logger.Log.Fatal("Error opening database connection", zap.Error(err))
			return
		}

		DB = db

		// Создание таблицы для хранения сокращённых URL, если она не существует.
		_, err = DB.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS short_urls (
				id SERIAL PRIMARY KEY,
				short_url TEXT NOT NULL UNIQUE,
				original_url TEXT NOT NULL UNIQUE,
			    user_id TEXT NOT NULL,
			    is_deleted BOOLEAN NOT NULL
			)
		`)
		if err != nil {
			logger.Log.Error("Error creating table", zap.Error(err))
			return
		}

		// Загрузка существующих URL из базы данных.
		err = loadURLsFromDB()
		if err != nil {
			logger.Log.Error("Error loading URLs from DB", zap.Error(err))
			return
		}

	} else if config.FileStoragePath != "" {
		// Загрузка существующих URL из файлового хранилища.
		err := loadURLsFromFile()
		if err != nil {
			logger.Log.Error("Error loading URLs from file", zap.Error(err))
			return
		}
	}
}

// loadURLsFromDB загружает данные сокращённых URL из базы данных в память.
func loadURLsFromDB() error {
	rows, err := DB.Query("SELECT short_url, original_url FROM short_urls")
	if err != nil {
		return err
	}
	defer rows.Close()

	// Преобразуем каждую запись в мапу.
	for rows.Next() {
		var shortURL, originalURL string
		if err = rows.Scan(&shortURL, &originalURL); err != nil {
			return err
		}
		URLStore[shortURL] = originalURL
	}
	return rows.Err()
}

// loadURLsFromFile загружает данные сокращённых URL из файла в память.
func loadURLsFromFile() error {
	consumer, err := file.NewConsumer(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer consumer.File.Close()

	// Чтение каждого события из файла и добавление его в мапу.
	for {
		event, err := consumer.ReadEvent()
		if err != nil {
			break
		}
		URLStore[event.ShortURL] = event.OriginalURL
	}
	return nil
}

// SaveURL сохраняет новый или обновлённый сокращённый URL в хранилище.
// В случае использования базы данных данные записываются в таблицу,
// в случае файлового хранилища — в файл. Возвращает сокращённый URL или ошибку,
// если URL уже существует.
func SaveURL(event *models.URLData) (string, error) {
	if DB != nil {

		// Проверяем, существует ли уже сокращённый URL для данного оригинального URL.
		err := DB.QueryRow(`
			SELECT short_url FROM short_urls WHERE original_url=$1
		`, event.OriginalURL).Scan(&ExistingShortURL)

		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
			} else {
				return "", err
			}
		}

		if ExistingShortURL != "" {
			event.ShortURL = ExistingShortURL
			return ExistingShortURL, ErrAlreadyExists
		}

		// Вставка нового URL в таблицу, с обновлением в случае конфликта.
		_, err = DB.Exec(`
			INSERT INTO short_urls (short_url, original_url, user_id, is_deleted) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (original_url)
			DO UPDATE SET short_url = short_urls.short_url
		`, event.ShortURL, event.OriginalURL, event.UserUUID, event.DeletedFlag)

		if err != nil {
			return "", err
		}

		return "", nil
	} else if config.FileStoragePath != "" {
		// Сохранение URL в файл при использовании файлового хранилища.
		producer, err := file.NewProducer(config.FileStoragePath)
		if err != nil {
			return "", err
		}
		defer producer.File.Close()

		// Блокировка для синхронизации работы с памятью.
		Mu.Lock()
		URLStore[event.ShortURL] = event.OriginalURL
		Mu.Unlock()

		return "", producer.WriteEvent(event)
	}

	// Сохранение URL в память, если нет базы данных или файлового хранилища.
	Mu.Lock()
	URLStore[event.ShortURL] = event.OriginalURL
	Mu.Unlock()
	return "", nil
}

// GetOriginalURL возвращает оригинальный URL для сокращённого URL,
// а также флаг, указывающий, был ли он удалён.
func GetOriginalURL(shortID string) (string, bool, bool) {
	if DB != nil {
		var originalURL string
		var deleted bool
		err := DB.QueryRow("SELECT original_url, is_deleted FROM short_urls WHERE short_url = $1", shortID).Scan(&originalURL, &deleted)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", false, false
			}
			return "", false, false
		}
		return originalURL, deleted, true
	}

	// Получение URL из памяти, если нет базы данных.
	Mu.Lock()
	defer Mu.Unlock()
	originalURL, ok := URLStore[shortID]
	return originalURL, false, ok
}

// GetURLsByUser возвращает все сокращённые URL для указанного пользователя.
func GetURLsByUser(userID string) ([]models.URLData, error) {
	if DB != nil {
		rows, err := DB.Query("SELECT short_url, original_url FROM short_urls WHERE user_id = $1", userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var urls []models.URLData
		for rows.Next() {
			var shortURL, originalURL string
			if err := rows.Scan(&shortURL, &originalURL); err != nil {
				return nil, err
			}
			urls = append(urls, models.URLData{ShortURL: config.FlagBaseURL + "/" + shortURL, OriginalURL: originalURL})
		}
		return urls, rows.Err()
	}
	return nil, nil
}

// BatchUpdateDeleteFlag обновляет флаг is_deleted для указанного сокращённого URL,
// если он принадлежит указанному пользователю.
func BatchUpdateDeleteFlag(urlID string, userID string) error {
	query := `UPDATE short_urls SET is_deleted = TRUE WHERE short_url = $1 AND user_id = $2`
	_, err := DB.Exec(query, urlID, userID)
	return err
}
