package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sol1corejz/go-url-shortener/internal/models"
)

func BenchmarkHandlePost(b *testing.B) {
	requestBody := []byte("https://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(requestBody))
		req.Header.Set("Content-Type", "text/plain")

		w := httptest.NewRecorder()
		HandlePost(w, req)

		if w.Code != http.StatusCreated {
			b.Errorf("unexpected status code: got %d, want %d", w.Code, http.StatusCreated)
		}
	}
}

func BenchmarkHandleJSONPost(b *testing.B) {

	requestBody := []byte(`{"url": "https://example.com"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/shorten", io.NopCloser(bytes.NewBuffer(requestBody)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		response := w.Result()
		defer response.Body.Close()

		HandleJSONPost(w, req)

		if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusConflict && response.StatusCode != http.StatusOK {
			b.Errorf("Unexpected status code: %d", response.StatusCode)
		}

	}
}

func BenchmarkHandleBatchPost(b *testing.B) {
	batchRequest := []models.BatchRequest{
		{OriginalURL: "http://example.com/1", CorrelationID: "cor1"},
		{OriginalURL: "http://example.com/2", CorrelationID: "cor2"},
		{OriginalURL: "http://example.com/3", CorrelationID: "cor3"},
	}

	requestBody, err := json.Marshal(batchRequest)
	if err != nil {
		b.Fatalf("failed to marshal request body: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/batch", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		HandleBatchPost(w, req)

		if w.Code != http.StatusCreated {
			b.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}
	}
}

// ExampleHandlePost демонстрирует использование обработчика HandlePost.
func ExampleHandlePost() {
	// Создаем HTTP-запрос с POST-методом.
	body := strings.NewReader("https://example.com")
	req, err := http.NewRequest(http.MethodPost, "/", body)
	if err != nil {
		panic(err)
	}

	// Создаем ResponseRecorder для записи ответа.
	rec := httptest.NewRecorder()

	// Вызов обработчика.
	HandlePost(rec, req)

	// Проверяем статус-код ответа.
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа.
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса.
	fmt.Println(resp.StatusCode)
	// Вывод тела ответа.
	fmt.Println(string(responseBody))

	// Output:
	// HTTP Status: 201
	// http://localhost/<ShortID>
}

// ExampleHandleJSONPost демонстрирует использование обработчика HandleJSONPost.
func ExampleHandleJSONPost() {
	// Подготовка данных запроса.
	requestData := models.Request{
		URL: "http://example.com",
	}
	body, _ := json.Marshal(requestData)

	// Создание HTTP-запроса.
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Создание ResponseRecorder для записи ответа.
	rec := httptest.NewRecorder()

	// Вызов обработчика.
	HandleJSONPost(rec, req)

	// Проверяем статус-код ответа.
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа.
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса.
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа.
	fmt.Println("Response Body:", responseBody)

	// Output:
	// HTTP Status: 201
	// Response Body: {"result":"http://localhost/<shortID>"}
}

// ExampleHandleBatchPost демонстрирует использование обработчика HandleBatchPost.
func ExampleHandleBatchPost() {
	// Подготовка данных запроса
	requestData := []models.BatchRequest{
		{OriginalURL: "http://example.com/1", CorrelationID: "1"},
		{OriginalURL: "http://example.com/2", CorrelationID: "2"},
	}
	body, _ := json.Marshal(requestData)

	// Создание HTTP-запроса
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Создание ResponseRecorder для записи ответа
	rec := httptest.NewRecorder()

	// Вызов обработчика
	HandleBatchPost(rec, req)

	// Проверяем статус-код ответа.
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа.
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа
	fmt.Println("Response Body:", responseBody)

	// Output:
	// HTTP Status: 201
	// Response Body: [{"correlation_id":"1","short_url":"http://localhost/short1"},{"correlation_id":"2","short_url":"http://localhost/short2"}]
}

// ExampleHandleDeleteURLs демонстрирует использование обработчика HandleDeleteURLs.
func ExampleHandleDeleteURLs() {
	// Подготовка тестовых данных
	idsToDelete := []string{"abc123", "xyz456"}
	body, _ := json.Marshal(idsToDelete)

	// Создание HTTP-запроса с методом DELETE и телом запроса
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Создание ResponseRecorder для записи ответа
	rec := httptest.NewRecorder()

	// Вызов обработчика
	HandleDeleteURLs(rec, req)

	// Проверяем статус-код ответа.
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа.
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа (в данном случае тело пустое)
	fmt.Println("Response Body:", responseBody)

	// Output:
	// HTTP Status: 202
	// Response Body:
}

// ExampleHandleGet демонстрирует использование обработчика HandleGet.
func ExampleHandleGet() {
	// Подготовка тестового запроса
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)

	// Создаём ResponseRecorder для записи ответа
	rec := httptest.NewRecorder()

	// Вызов обработчика
	HandleGet(rec, req)

	// Получаем результат
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа (в данном случае это редирект)
	fmt.Println("Response Body:", responseBody)

	// Output:
	// HTTP Status: 307
	// Response Body: https://example.com
}

// ExampleHandleGetUserURLs демонстрирует использование обработчика HandleGetUserURLs.
func ExampleHandleGetUserURLs() {
	// Создаём тестовый HTTP-запрос
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	// Устанавливаем заголовок авторизации (мокаем JWT токен)
	req.Header.Set("Authorization", "Bearer valid_token")

	// Создаём ResponseRecorder для записи ответа
	rec := httptest.NewRecorder()

	// Вызов обработчика
	HandleGetUserURLs(rec, req)

	// Получаем результат
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа
	fmt.Println("Response Body:", string(responseBody))

	// Output:
	// HTTP Status: 200
	// Response Body: ["https://example.com","https://test.com"]
}

// ExampleHandlePing демонстрирует использование обработчика HandlePing.
func ExampleHandlePing() {
	// Создаём тестовый HTTP-запрос
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	// Создаём ResponseRecorder для записи ответа
	rec := httptest.NewRecorder()

	// Вызов обработчика
	HandlePing(rec, req)

	// Получаем результат
	resp := rec.Result()
	defer resp.Body.Close()

	// Читаем тело ответа
	responseBody, _ := io.ReadAll(resp.Body)

	// Вывод HTTP-статуса
	fmt.Println("HTTP Status:", resp.StatusCode)

	// Вывод тела ответа
	fmt.Println("Response Body:", string(responseBody))

	// Output:
	// HTTP Status: 200
	// Response Body: pong
}
