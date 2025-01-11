package handlers

import (
	"context"
	"errors"
	"fmt"
	pb "github.com/sol1corejz/go-url-shortener/proto"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
)

// ShortenerServer представляет сервер для обработки gRPC-запросов.
// Включает методы, соответствующие gRPC-интерфейсу.
type ShortenerServer struct {
	pb.UnimplementedShortenerServer
}

// TimeOutErr ошибка времени выполнения
var TimeOutErr = errors.New("request timed out")

// SaveShortURL содержит бизнес-логику обработки и сохранения URL.
func SaveShortURL(ctx context.Context, originalURL, userID string) (string, error) {
	select {
	case <-ctx.Done():
		return "", TimeOutErr
	default:
		// Проверка на пустой URL
		if originalURL == "" {
			return "", errors.New("empty URL")
		}

		// Генерация короткого идентификатора
		shortID := generateShortID()
		shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

		// Создание структуры с данными для сохранения
		event := models.URLData{
			OriginalURL: originalURL,
			ShortURL:    shortID,
			UUID:        uuid.New().String(),
			UserUUID:    userID,
			DeletedFlag: false,
		}

		// Попытка сохранить URL в хранилище
		if existURL, err := storage.SaveURL(&event); err != nil {
			if errors.Is(err, storage.ErrAlreadyExists) {
				return fmt.Sprintf("%s/%s", config.FlagBaseURL, existURL), storage.ErrAlreadyExists
			}
			return "", err
		}

		return shortURL, nil
	}
}

// HandlePost обрабатывает POST-запрос, содержащий оригинальный URL, и генерирует для него короткий URL.
// Функция выполняет следующие действия:
// 1. Проверяет наличие токена в cookies. Если токен отсутствует, генерирует новый токен и устанавливает его в cookie.
// 2. Читает тело запроса, ожидая, что в нем будет содержаться оригинальный URL.
// 3. Генерирует короткий идентификатор для URL и формирует короткий URL.
// 4. Создает структуру данных с информацией о URL и сохраняет ее в хранилище.
// 5. Если URL уже существует, возвращает существующий короткий URL с кодом 409 (Conflict).
// 6. В случае успешного создания короткого URL, возвращает его с кодом 201 (Created).
//
// В случае ошибок возвращаются соответствующие HTTP-статусы:
// - 400 (Bad Request) для пустого URL,
// - 401 (Unauthorized) для невалидного токена,
// - 409 (Conflict) если короткий URL уже существует,
// - 500 (Internal Server Error) в случае проблем на сервере.
func HandlePost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Проверка наличия токена в cookie
	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		// Генерация нового токена
		token, err := auth.GenerateToken()
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
		userID = auth.GetUserID(token)
	} else if err != nil {
		http.Error(w, "Error retrieving cookie", http.StatusBadRequest)
		return
	} else {
		userID = auth.GetUserID(cookie.Value)
		if userID == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Чтение тела запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	originalURL := strings.TrimSpace(string(body))

	// Используем общую бизнес-логику
	shortURL, err := SaveShortURL(ctx, originalURL, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
		if errors.Is(err, TimeOutErr) {
			http.Error(w, "Request timed out", http.StatusRequestTimeout)
		}
		http.Error(w, "Failed to save URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// CreateShortURL обрабатывает gRPC-запрос для создания короткого URL.
func (s *ShortenerServer) CreateShortURL(ctx context.Context, req *pb.CreateShortURLRequest) (*pb.CreateShortURLResponse, error) {
	userID := req.UserId
	originalURL := req.OriginalUrl

	// Используем общую бизнес-логику
	shortURL, err := SaveShortURL(ctx, originalURL, userID)
	if err != nil {
		if errors.Is(err, storage.ErrAlreadyExists) {
			return &pb.CreateShortURLResponse{ShortUrl: shortURL, Error: "URL already exists"}, status.Errorf(http.StatusConflict, "URL already exists")
		}
		return &pb.CreateShortURLResponse{Error: "Internal server error"}, status.Errorf(http.StatusInternalServerError, "Internal server error")
	}

	return &pb.CreateShortURLResponse{ShortUrl: shortURL}, nil
}
