// Package logger предоставляет функции для инициализации и использования логирования
// в приложении, включая логирование HTTP-запросов с помощью библиотеки zap.
package logger

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Log является глобальной переменной для использования логгера. Изначально настроен на no-op логгер.
var Log = zap.NewNop()

// Initialize настраивает и инициализирует логгер с указанным уровнем логирования.
// Принимает строковый параметр level, который указывает уровень логирования, например: "info", "debug" и т.д.
// Возвращает ошибку, если уровень логирования некорректен или произошла ошибка при создании логгера.
func Initialize(level string) error {

	// Парсит уровень логирования.
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	// Создаёт конфигурацию логгера с настройками для production.
	cfg := zap.NewProductionConfig()

	// Устанавливает уровень логирования.
	cfg.Level = lvl

	// Создаёт новый логгер с заданной конфигурацией.
	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	// Устанавливает глобальный логгер.
	Log = zl
	return nil
}

// RequestLogger оборачивает HTTP-обработчик, логируя информацию о запросах.
// Принимает HTTP-обработчик, возвращает новый обработчик, который записывает в лог информацию
// о пути запроса, методе и времени выполнения запроса.
func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		// Сохраняем время начала обработки запроса.
		start := time.Now()

		// Извлекаем путь и метод запроса.
		uri := r.RequestURI
		method := r.Method

		// Выполняем оригинальный обработчик.
		h(w, r)

		// Вычисляем длительность запроса.
		duration := time.Since(start)

		// Записываем информацию о запросе в лог.
		Log.Info("got incoming HTTP request",
			zap.String("path", uri),
			zap.String("method", method),
			zap.String("duration", strconv.FormatInt(int64(duration), 10)),
		)

	}

	return logFn
}
