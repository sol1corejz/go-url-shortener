package config

import (
	"flag"
	"os"
)

var (
	FlagRunAddr     string
	FlagBaseURL     string
	FlagLogLevel    string
	FileStoragePath string
	DefaultFilePath = "urls.json"
	DatabaseDSN     string
	DefaultDBDSN    = "localhost"
)

func ParseFlags() {

	flag.StringVar(&FlagRunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&FlagBaseURL, "b", "http://localhost:8080", "base URL for shortened links")
	flag.StringVar(&FlagLogLevel, "l", "info", "log level")
	flag.StringVar(&FileStoragePath, "f", "", "file storage path")
	flag.StringVar(&DatabaseDSN, "d", "", "databse dsn")
	flag.Parse()

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

	if FileStoragePath == "" {
		FileStoragePath = DefaultFilePath
	}

	if DatabaseDSN == "" {
		DatabaseDSN = DefaultDBDSN
	}
}
