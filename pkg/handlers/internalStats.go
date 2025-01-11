package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	pb "github.com/sol1corejz/go-url-shortener/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
	"net/http"
)

// ErrFailedToCount - ошибка подсчета из бд
var ErrFailedToCount = errors.New("failed to count error")

// GetStats получает информацию из бд
func GetStats() (int, int, error) {
	// Получаем количество сокращённых URL
	countURLs, err := storage.GetURLsCount()
	if err != nil {
		logger.Log.Error("Failed to count URLs", zap.Error(err))
		return 0, 0, ErrFailedToCount
	}

	// Получаем количество пользователей
	countUsers, err := storage.GetUsersCount()
	if err != nil {
		logger.Log.Error("Failed to count users", zap.Error(err))
		return 0, 0, ErrFailedToCount
	}

	return countURLs, countUsers, nil
}

// HandleGetInternalStats обрабатывает запрос на получение статистики.
// Количество сокращенных URL и количество уникальных пользователей
// В случае ошибки возвращает соответствующий статус.
func HandleGetInternalStats(w http.ResponseWriter, r *http.Request) {
	countURLs, countUsers, err := GetStats()
	// Если произошла ошибка при получении данных, возвращаем ошибку 500 (Internal Server Error).
	if err != nil {
		http.Error(w, "Failed to count stats", http.StatusInternalServerError)
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

// GetInternalStats обрабатывает gRPC-запрос для получения статистики.
func (s *ShortenerServer) GetInternalStats(ctx context.Context, req *pb.GetInternalStatsRequest) (*pb.GetInternalStatsResponse, error) {
	countURLs, countUsers, err := GetStats()

	if err != nil {
		return &pb.GetInternalStatsResponse{
			Error: "Failed to count stats",
		}, status.Error(http.StatusInternalServerError, "Failed to count stats")
	}

	return &pb.GetInternalStatsResponse{
		Urls:  int32(countURLs),
		Users: int32(countUsers),
	}, nil
}
