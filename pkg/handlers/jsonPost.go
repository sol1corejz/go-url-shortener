package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	pb "github.com/sol1corejz/go-url-shortener/proto"
	"google.golang.org/grpc/status"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

// HandleJSONPost обрабатывает POST-запрос с JSON-данными, содержащими URL.
// Функция выполняет следующие действия:
// 1. Проверяет наличие токена в cookies. Если токен отсутствует, генерирует новый токен и устанавливает его в cookie.
// 2. Декодирует JSON-данные из тела запроса и извлекает URL.
// 3. Генерирует короткий URL, связывая его с оригинальным URL.
// 4. Сохраняет данные URL в хранилище. Если URL уже существует, возвращает существующий короткий URL с кодом 409 (Conflict).
// 5. Если все прошло успешно, возвращает короткий URL в формате JSON с кодом 201 (Created).
//
// В случае ошибок возвращаются соответствующие HTTP-статусы, например, 400 (Bad Request) при неверных данных или 500 (Internal Server Error)
// при проблемах с сервером.
func HandleJSONPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Проверка наличия токена в cookie.
	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		var token string
		// Если токен отсутствует, генерируется новый токен и устанавливается в cookie.
		token, err = auth.GenerateToken()
		if err != nil {
			http.Error(w, "Unable to generate token", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  time.Now().Add(auth.TokenExp),
			HttpOnly: true,
		})

		// Получаем идентификатор пользователя из токена.
		userID = auth.GetUserID(token)
	} else if err != nil {
		http.Error(w, "Error retrieving cookie", http.StatusBadRequest)
		return
	} else {
		// Получаем идентификатор пользователя из cookie.
		userID = auth.GetUserID(cookie.Value)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Декодирование тела запроса в структуру models.Request.
	var req models.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Проверка наличия URL в запросе.
	if req.URL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	// Генерация короткого идентификатора и формирования короткого URL.
	shortID := generateShortID()
	shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

	// Подготовка ответа.
	resp := models.Response{
		Result: shortURL,
	}

	// Создание объекта с данными для сохранения.
	event := models.URLData{
		OriginalURL: req.URL,
		ShortURL:    shortID,
		UUID:        uuid.New().String(),
		UserUUID:    userID,
		DeletedFlag: false,
	}

	// Ожидание завершения операции сохранения URL или тайм-аута.
	select {
	case <-ctx.Done():
		http.Error(w, "Request canceled or timed out", http.StatusRequestTimeout)
		return
	default:
		// Попытка сохранить URL в хранилище.
		if existURL, err := storage.SaveURL(&event); err != nil {

			// Если URL уже существует, возвращаем существующий короткий URL с кодом 409.
			if errors.Is(err, storage.ErrAlreadyExists) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)

				resp = models.Response{
					Result: fmt.Sprintf("%s/%s", config.FlagBaseURL, existURL),
				}
				json.NewEncoder(w).Encode(resp)

				storage.ExistingShortURL = ""
				return
			}

			// Ошибка при сохранении URL.
			http.Error(w, "Failed to save URL", http.StatusInternalServerError)
			return
		}
	}

	// Устанавливаем заголовок и возвращаем успешный ответ с созданным коротким URL.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Кодирование и отправка ответа в формате JSON.
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Log.Debug("error encoding response", zap.Error(err))
		return
	}
}

// CreateJSONShortURL обрабатывает gRPC-запрос для создания короткого URL из JSON-запроса.
func (s *ShortenerServer) CreateJSONShortURL(ctx context.Context, req *pb.CreateJSONShortURLRequest) (*pb.CreateJSONShortURLResponse, error) {
	userID := req.UserId
	originalURL := req.OriginalUrl

	// Проверка на пустой URL.
	if originalURL == "" {
		return &pb.CreateJSONShortURLResponse{
			Error: "Empty URL",
		}, status.Error(http.StatusBadRequest, "Empty URL")
	}

	// Используем общую бизнес-логику для сохранения URL.
	shortURL, err := SaveShortURL(ctx, originalURL, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return &pb.CreateJSONShortURLResponse{
				ShortUrl: fmt.Sprintf("%s/%s", config.FlagBaseURL, shortURL),
				Error:    "URL already exists",
			}, status.Error(http.StatusBadRequest, "URL already exists")
		}
		if errors.Is(err, TimeOutErr) {
			return &pb.CreateJSONShortURLResponse{
				Error: "Request timed out",
			}, status.Error(http.StatusRequestTimeout, "Request timed out")
		}
		return &pb.CreateJSONShortURLResponse{
			Error: "Failed to save URL",
		}, status.Error(http.StatusInternalServerError, "Failed to save URL")
	}

	// Возвращаем успешный ответ.
	return &pb.CreateJSONShortURLResponse{
		ShortUrl: shortURL,
	}, nil
}
