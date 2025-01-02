// Package config отвечает за чтение конфигурации приложения из переменных окружения и флагов.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

// Структура для хранения конфигурации из JSON-файла.
type Config struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
	TrustedSubnet   string `json:"trusted_subnet"`
}

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
	EnableHTTPS bool
	// ConfigFilePath содержит путь до файла конфигурации
	ConfigFilePath string
	// TrustedSubnet добавляет проверку, что переданный IP-адрес клиента входит в доверенную подсеть
	TrustedSubnet string
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
	flag.BoolVar(&EnableHTTPS, "s", false, "connection type")
	flag.StringVar(&ConfigFilePath, "c", "", "path to configuration JSON file")
	flag.StringVar(&TrustedSubnet, "t", "", "trusted subnet check")
	flag.Parse()

	// Чтение значений из файла конфигурации, если он указан.
	if configPath := os.Getenv("CONFIG"); configPath != "" {
		configData, err := loadConfig(configPath)
		if err != nil {
			log.Printf("Warning: failed to read config file: %v", err)
		}

		FlagRunAddr = configData.ServerAddress
		FlagBaseURL = configData.BaseURL
		FileStoragePath = configData.FileStoragePath
		DatabaseDSN = configData.DatabaseDSN
		EnableHTTPS = configData.EnableHTTPS
		TrustedSubnet = configData.TrustedSubnet
	}

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
		EnableHTTPS = true
	}

	if trustedSubnet := os.Getenv("TRUSTED_SUBNET"); trustedSubnet != "" {
		TrustedSubnet = trustedSubnet
	}
}

// функция загрузки конфига из файла
func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
