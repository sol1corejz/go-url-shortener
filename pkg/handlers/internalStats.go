package handlers

import (
	"encoding/json"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
	"net/http"
)

// HandleGetInternalStats обрабатывает запрос на получение статистики.
// Количество сокращенных URL и количество уникальных пользователей
// В случае ошибки возвращает соответствующий статус.
func HandleGetInternalStats(w http.ResponseWriter, r *http.Request) {
	// Получаем количество сокращённых URL
	countURLs, err := storage.GetURLsCount()
	if err != nil {
		// Если произошла ошибка при получении данных, возвращаем ошибку 500 (Internal Server Error).
		logger.Log.Error("Failed to count URLs", zap.Error(err))
		http.Error(w, "Failed to count URLs", http.StatusInternalServerError)
		return
	}

	// Получаем количество пользователей
	countUsers, err := storage.GetUsersCount()
	if err != nil {
		// Если произошла ошибка при получении данных, возвращаем ошибку 500 (Internal Server Error).
		logger.Log.Error("Failed to count users", zap.Error(err))
		http.Error(w, "Failed to count users", http.StatusInternalServerError)
		return
	}

	stats := models.InternalStatsResponse{
		URLs:  countURLs,
		Users: countUsers,
	}

	// Устанавливаем заголовок ответа для JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Кодируем статистику в JSON и отправляем в ответ.
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		// Если не удалось закодировать ответ в JSON, логируем ошибку и возвращаем ошибку 500 (Internal Server Error).
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
