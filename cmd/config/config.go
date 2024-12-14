// Package config отвечает за чтение конфигурации приложения из переменных окружения и флагов.
package config

import (
	"flag"
	"os"
)

// Переменные для хранения значений env и флагов.
var (
	// FlagRunAddr содержит адрес и порт для запуска сервера.
	FlagRunAddr string
	// FlagBaseURL содержит базовый URL для сокращенных ссылок.
	FlagBaseURL string
	// FlagLogLevel задает уровень логирования приложения.
	FlagLogLevel string
	// FileStoragePath определяет путь к файлу для хранения данных.
	FileStoragePath string
	// DatabaseDSN содержит строку подключения к базе данных.
	DatabaseDSN string
	// EnableHTTPS определяет тип соединения.
	EnableHTTPS string
)

// ParseFlags читает флаги командной строки и переменные окружения.
// Если указаны как флаги, так и переменные окружения, приоритет имеют значения из переменных окружения.
func ParseFlags() {
	// Инициализация флагов командной строки.
	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&FlagBaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.StringVar(&FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&DatabaseDSN, "d", "", "databse dsn")
	flag.StringVar(&EnableHTTPS, "s", "", "connection type")
	flag.Parse()

	// Переопределение значений флагов переменными окружения (если они заданы).
	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		FlagRunAddr = envRunAddr
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		FlagBaseURL = envBaseURL
	}

	if envStoragePath := os.Getenv("FILE_STORAGE_PATH"); envStoragePath != "" {
		FileStoragePath = envStoragePath
	}

	if databaseDsn := os.Getenv("DATABASE_DSN"); databaseDsn != "" {
		DatabaseDSN = databaseDsn
	}

	if enableHTTPS := os.Getenv("ENABLE_HTTPS"); enableHTTPS != "" {
		EnableHTTPS = enableHTTPS
	}
}
