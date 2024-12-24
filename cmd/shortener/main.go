// Модуль main — входная точка приложения, где происходит инициализация конфигурации, хранилища и запуск HTTP-сервера.
package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/cert"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"github.com/sol1corejz/go-url-shortener/pkg/handlers"
	"go.uber.org/zap"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
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
	// Канал сообщения о закртии соединения
	idleConnsClosed := make(chan struct{})
	// Канал для перенаправления прерываний
	sigint := make(chan os.Signal, 1)
	// Регистрация прерываний
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	//Контекст отмены
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Вывод информации о версии сборки.
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	// Считывает флаги конфигурации и обновляет параметры запуска.
	config.ParseFlags()

	// Инициализирует хранилище на основе параметров конфигурации.
	storage.InitializeStorage(ctx)

	// Запускает сервер, передавая канал `sigint` для обработки сигналов.
	if err := run(ctx, sigint, idleConnsClosed); err != nil {
		logger.Log.Error("Failed to run server", zap.Error(err))
	}

	<-idleConnsClosed
	// Сообщение о закрытии соединения
	logger.Log.Info("Server Shutdown gracefully")
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
func run(ctx context.Context, sigint chan os.Signal, idleConnsClosed chan struct{}) error {
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

	// Создаем сервер
	srv := &http.Server{
		Addr:    config.FlagRunAddr,
		Handler: r,
	}

	// Горутину для обработки пойманных прерываний
	go func() {
		// читаем из канала прерываний
		<-sigint
		// получили сигнал os.Interrupt, запускаем процедуру graceful shutdown
		if err := srv.Shutdown(ctx); err != nil {
			// ошибки закрытия Listener
			log.Printf("HTTP server Shutdown: %v", err)
		}
		// сообщаем основному потоку,
		// что все сетевые соединения обработаны и закрыты
		close(idleConnsClosed)

	}()

	// Запускает HTTP-сервер на заданном адресе и типе подключения.
	if config.EnableHTTPS {
		if !cert.CertExists() {
			logger.Log.Info("Generating new TLS certificate")
			certPEM, keyPEM := cert.GenerateCert()
			if err := cert.SaveCert(certPEM, keyPEM); err != nil {
				return fmt.Errorf("failed to save TLS certificate: %w", err)
			}
		}

		logger.Log.Info("Loading existing TLS certificate")
		return http.ListenAndServeTLS(config.FlagRunAddr, cert.CertificateFilePath, cert.KeyFilePath, r)
	}
	return http.ListenAndServe(config.FlagRunAddr, r)
}
