package handlers

import (
	"context"
	"errors"
	"fmt"
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

	// Проверка наличия токена в cookie.
	cookie, err := r.Cookie("token")
	var userID string
	if errors.Is(err, http.ErrNoCookie) {
		// Если токен отсутствует, генерируется новый токен и устанавливается в cookie.
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

	// Чтение тела запроса.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Преобразование тела в строку и проверка, что URL не пустой.
	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	// Генерация короткого идентификатора для URL.
	shortID := generateShortID()
	shortURL := fmt.Sprintf("%s/%s", config.FlagBaseURL, shortID)

	// Создание структуры с данными для сохранения.
	event := models.URLData{
		OriginalURL: originalURL,
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
				w.Write([]byte(fmt.Sprintf("%s/%s", config.FlagBaseURL, existURL)))

				storage.ExistingShortURL = ""
				return
			}

			// Ошибка при сохранении URL.
			http.Error(w, "Failed to save URL", http.StatusInternalServerError)
			return
		}
	}

	// Устанавливаем заголовок и возвращаем успешный ответ с созданным коротким URL.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
