// Пакет handlers содержит обработчики HTTP-запросов для сервиса сокращения URL.
package handlers

import (
	"context"
	"encoding/json"
	pb "github.com/sol1corejz/go-url-shortener/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"net/http"
	"sync"

	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"go.uber.org/zap"
)

// HandleDeleteURLs обрабатывает запросы на удаление списка сокращённых URL.
// Проверяет авторизацию пользователя, извлекает список идентификаторов из тела запроса
// и инициирует асинхронный процесс удаления.
//
// Поддерживаемый метод HTTP: DELETE
// Тело запроса: JSON-массив идентификаторов сокращённых URL (например, ["abc123", "xyz456"]).
// Ответы:
// - 202 Accepted: Удаление батча начато.
// - 401 Unauthorized: Пользователь не авторизован.
// - 400 Bad Request: Неверный формат JSON или пустой батч.
// - 500 Internal Server Error: Ошибка чтения тела запроса.
func HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	// Проверка авторизации пользователя с помощью функции CheckIsAuthorized.
	// Если авторизация не пройдена, возвращаем ошибку 401 (Unauthorized).
	userID, err := auth.CheckIsAuthorized(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Чтение тела запроса для получения массива идентификаторов URL, которые нужно удалить.
	var ids []string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// Если произошла ошибка при чтении тела запроса, возвращаем ошибку 500 (Internal Server Error).
		http.Error(w, "Не удалось прочитать тело запроса", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close() // Закрываем тело запроса после его чтения.

	// Попытка декодирования тела запроса из JSON в срез строк (идентификаторы URL).
	if err = json.Unmarshal(body, &ids); err != nil {
		// Если JSON не удалось декодировать, логируем ошибку и возвращаем ошибку 400 (Bad Request).
		logger.Log.Info("Не удалось декодировать JSON батча", zap.Error(err))
		http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	// Проверка, что батч не пустой. Если батч пустой, возвращаем ошибку 400 (Bad Request).
	if len(ids) == 0 {
		http.Error(w, "Батч не может быть пустым", http.StatusBadRequest)
		return
	}

	// Устанавливаем код ответа 202 (Accepted), так как процесс удаления будет выполнен асинхронно.
	w.WriteHeader(http.StatusAccepted)

	// Запуск асинхронного процесса удаления URL.
	// В процессе удаления будет использован список идентификаторов и идентификатор пользователя.
	go processDeleteBatch(ids, userID)
}

func processDeleteBatch(ids []string, userID string) {
	doneCh := make(chan struct{})
	defer close(doneCh)

	inputCh := generatorDeleteBatch(doneCh, ids)
	channels := fanOutDeleteBatch(doneCh, inputCh, userID)
	errorCh := fanInDeleteBatch(doneCh, channels...)

	for err := range errorCh {
		if err != nil {
			logger.Log.Error("Не удалось удалить URL", zap.Error(err))
		}
	}
}

func deleteURL(doneCh chan struct{}, inputCh chan string, userID string) chan error {
	resultCh := make(chan error)
	go func() {
		defer close(resultCh)
		for id := range inputCh {
			err := storage.BatchUpdateDeleteFlag(id, userID)
			select {
			case <-doneCh:
				return
			case resultCh <- err:
			}
		}
	}()
	return resultCh
}

func generatorDeleteBatch(doneCh chan struct{}, ids []string) chan string {
	inputCh := make(chan string)
	go func() {
		defer close(inputCh)
		for _, id := range ids {
			select {
			case <-doneCh:
				return
			case inputCh <- id:
			}
		}
	}()
	return inputCh
}

func fanOutDeleteBatch(doneCh chan struct{}, inputCh chan string, userID string) []chan error {
	numWorkers := 5
	channels := make([]chan error, numWorkers)
	for i := 0; i < numWorkers; i++ {
		channels[i] = deleteURL(doneCh, inputCh, userID)
	}
	return channels
}

func fanInDeleteBatch(doneCh chan struct{}, resultChs ...chan error) chan error {
	finalCh := make(chan error)
	var wg sync.WaitGroup

	for _, ch := range resultChs {
		wg.Add(1)

		chClosure := ch

		go func() {
			defer wg.Done()

			for err := range chClosure {
				select {
				case <-doneCh:
					return
				case finalCh <- err:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(finalCh)
	}()

	return finalCh
}

// BatchDelete обрабатывает gRPC-запрос на удаление списка сокращённых URL.
func (s *ShortenerServer) BatchDelete(ctx context.Context, req *pb.BatchDeleteRequest) (*pb.BatchDeleteResponse, error) {
	userID := req.UserId

	// Проверка, что список идентификаторов не пустой.
	if len(req.Ids) == 0 {
		return &pb.BatchDeleteResponse{
			Error: "Batch cannot be empty",
		}, status.Error(codes.InvalidArgument, "Batch cannot be empty")
	}

	// Запуск асинхронного процесса удаления.
	go processDeleteBatch(req.Ids, userID)

	// Возврат успешного ответа.
	return &pb.BatchDeleteResponse{
		Message: "Batch deletion started",
	}, nil
}
