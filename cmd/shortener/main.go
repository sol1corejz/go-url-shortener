// Модуль main — входная точка приложения, где происходит инициализация конфигурации, хранилища и запуск HTTP-сервера.
package main

import (
	"context"
	"fmt"
	"github.com/sol1corejz/go-url-shortener/internal/cert"
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"github.com/sol1corejz/go-url-shortener/pkg/handlers"
	"go.uber.org/zap"
)

// Глобальные переменные для информации о версии сборки.
var (
	buildVersion = "N/A" // Версия сборки, передается на этапе компиляции.
	buildDate    = "N/A" // Дата сборки, передается на этапе компиляции.
	buildCommit  = "N/A" // Коммит сборки, передается на этапе компиляции.
)

// main — основная функция, которая запускает приложение.
// Здесь производится обработка флагов конфигурации, инициализация хранилища и вызов функции запуска сервера.
func main() {
	// Вывод информации о версии сборки.
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	// Считывает флаги конфигурации и обновляет параметры запуска.
	config.ParseFlags()

	// Создаёт контекст для управления жизненным циклом приложения.
	ctx := context.Background()

	// Инициализирует хранилище на основе параметров конфигурации.
	storage.InitializeStorage(ctx)

	// Запускает сервер. Если возникает ошибка, приложение завершает работу.
	if err := run(); err != nil {
		logger.Log.Error("Failed to run server", zap.Error(err))
	}
}

// run запускает HTTP-сервер, определяет маршруты и подключает middleware.
// Если запуск сервера завершается с ошибкой, функция возвращает её.
//
// Маршруты:
// - "/" (POST): Обработчик для создания коротких URL.
// - "/{shortURL}" (GET): Обработчик для редиректа по короткому URL.
// - "/api/shorten" (POST): Обработчик для JSON-запросов на сокращение URL.
// - "/api/shorten/batch" (POST): Обработчик для пакетного сокращения URL.
// - "/api/user/urls" (GET): Обработчик для получения URL текущего пользователя.
// - "/api/user/urls" (DELETE): Обработчик для удаления списка URL.
// - "/ping" (GET): Обработчик для проверки доступности сервера.
//
// Middleware:
// - GzipMiddleware: Сжатие/распаковка данных для оптимизации запросов.
// - RequestLogger: Логирование каждого входящего запроса.
func run() error {
	// Инициализирует логгер с заданным уровнем логирования.
	if err := logger.Initialize(config.FlagLogLevel); err != nil {
		return err
	}

	logger.Log.Info("Running server", zap.String("address", config.FlagRunAddr))

	// Создаёт роутер с использованием библиотеки chi.
	r := chi.NewRouter()

	// Подключает обработчики для профилирования через пакет pprof.
	r.Mount("/debug/pprof", http.StripPrefix("/debug/pprof", http.HandlerFunc(pprof.Index)))
	r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	r.Handle("/debug/pprof/heap", http.HandlerFunc(pprof.Index))

	// Определяет основные маршруты для обработки запросов.
	r.Route("/", func(r chi.Router) {
		r.Post("/", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandlePost)))
		r.Get("/{shortURL}", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleGet)))
	})

	// Определяет маршруты для API.
	r.Route("/api", func(r chi.Router) {
		r.Post("/shorten", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleJSONPost)))
		r.Post("/shorten/batch", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleBatchPost)))
		r.Get("/user/urls", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleGetUserURLs)))
		r.Delete("/user/urls", logger.RequestLogger(middlewares.GzipMiddleware(handlers.HandleDeleteURLs)))
	})

	// Добавляет маршрут для проверки доступности сервера.
	r.Get("/ping", logger.RequestLogger(handlers.HandlePing))

	// Запускает HTTP-сервер на заданном адресе и типе подключения.
	if config.EnableHTTPS != "" {
		certPEM, privateKeyPEM := cert.GenerateCert()
		return http.ListenAndServeTLS(config.FlagRunAddr, certPEM, privateKeyPEM, r)
	}
	return http.ListenAndServe(config.FlagRunAddr, r)
}
