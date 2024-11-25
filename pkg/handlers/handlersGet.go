package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

func generateShortID() string {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// HandleGet обрабатывает запрос на получение оригинального URL по короткому идентификатору.
// При получении запроса с коротким URL, сервер проверяет его существование
// в хранилище и выполняет редирект на оригинальный URL, если он существует
// и не был удалён. В случае ошибки возвращает соответствующий статус.
func HandleGet(w http.ResponseWriter, r *http.Request) {

	// Извлекаем короткий URL из параметров запроса.
	id := chi.URLParam(r, "shortURL")
	if id == "" {
		// Если короткий URL не передан, возвращаем ошибку 400 (Bad Request).
		http.Error(w, "Invalid URL ID", http.StatusBadRequest)
		return
	}

	// Получаем оригинальный URL, флаг удаления и статус существования из хранилища.
	originalURL, deleted, ok := storage.GetOriginalURL(id)

	if !ok {
		// Если URL не найден, возвращаем ошибку 404 (Not Found).
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	// Если URL был удалён, возвращаем ошибку 410 (Gone).
	if deleted {
		http.Error(w, "URL deleted", http.StatusGone)
		return
	}

	// Если URL существует и не был удалён, выполняем редирект на оригинальный URL.
	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(originalURL))
}

// HandleGetUserURLs обрабатывает запрос на получение всех URL-адресов,
// сокращённых пользователем, который прошёл аутентификацию. В случае успешного
// запроса возвращает список URL в формате JSON. В случае отсутствия URL
// или ошибки возвращаются соответствующие статусы.
func HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {

	// Проверяем, авторизован ли пользователь.
	userID, err := auth.CheckIsAuthorized(r)

	if err != nil {
		// Если пользователь не авторизован, возвращаем ошибку 401 (Unauthorized).
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем список URL, сокращённых пользователем, из хранилища.
	urls, err := storage.GetURLsByUser(userID)
	if err != nil {
		// Если произошла ошибка при получении данных, возвращаем ошибку 500 (Internal Server Error).
		http.Error(w, "Failed to retrieve URLs", http.StatusInternalServerError)
		return
	}

	// Если у пользователя нет сокращённых URL, возвращаем статус 204 (No Content).
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Устанавливаем заголовок ответа для JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Кодируем список URL в JSON и отправляем в ответ.
	if err := json.NewEncoder(w).Encode(urls); err != nil {
		// Если не удалось закодировать ответ в JSON, логируем ошибку и возвращаем ошибку 500 (Internal Server Error).
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandlePing обрабатывает запрос на проверку состояния базы данных.
// Если подключение к базе данных работает, возвращает статус 200 OK с ответом "pong".
// В случае ошибки подключения возвращается статус 500.
func HandlePing(w http.ResponseWriter, r *http.Request) {
	// Пингует базу данных для проверки её состояния.
	if err := storage.DB.Ping(); err != nil {
		// Если ошибка подключения, возвращаем ошибку 500 (Internal Server Error).
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	// Если подключение успешно, возвращаем статус 200 OK и сообщение "pong".
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}
