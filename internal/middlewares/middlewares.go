// Package middlewares содержит промежуточные обработчики (middleware), которые
// выполняются во время обработки HTTP-запросов и отвечают за различные функциональности,
// такие как сжатие данных через Gzip.
package middlewares

import (
	"context"
	"github.com/sol1corejz/go-url-shortener/internal/auth"
	"github.com/sol1corejz/go-url-shortener/internal/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"net/http"
	"strings"

	"github.com/sol1corejz/go-url-shortener/cmd/gzip"
)

// GzipMiddleware — это промежуточный обработчик (middleware), который проверяет,
// поддерживает ли клиент сжатие данных с использованием Gzip, и если поддерживает,
// применяет сжатие для ответа. Если же запрос содержит сжатые данные, то он их
// распаковывает перед передачей в следующий обработчик.
//
// Этот middleware автоматически сжимает данные для клиентов, которые поддерживают
// Gzip, и распаковывает данные для запросов, которые отправляются с сжатыми данными.
//
// h — это исходный HTTP-обработчик, который будет вызван после обработки сжатия данных.
func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		// Проверяем, поддерживает ли клиент сжатие Gzip.
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip {
			// Если поддерживает, создаём новый сжимающий writer.
			cw := gzip.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		// Проверяем, сжаты ли данные в запросе.
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// Если запрос содержит сжатые данные, распаковываем их.
			cr, err := gzip.NewCompressReader(r.Body)
			if err != nil {
				// В случае ошибки при распаковке возвращаем ошибку 500.
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		// Вызываем исходный обработчик.
		h.ServeHTTP(ow, r)
	}
}

// TrustedSubnetMiddleware проверяет, входит ли IP клиента в доверенную подсеть.
func TrustedSubnetMiddleware(subnet string, h http.HandlerFunc) http.HandlerFunc {

	// Парсим подсеть на этапе создания middleware.
	_, trustedNet, err := net.ParseCIDR(subnet)
	if err != nil {
		// Если подсеть некорректна, запрещаем доступ к эндпоинту.
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Forbidden", http.StatusForbidden)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем IP клиента из заголовка.
		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Парсим IP клиента.
		ip := net.ParseIP(clientIP)
		if ip == nil || !trustedNet.Contains(ip) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Вызываем исходный обработчик.
		h.ServeHTTP(w, r)
	}
}

// AuthInterceptor проверяет наличие и валидность токена только для определённых методов.
func AuthInterceptor(protectedMethods []string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Проверяем, требует ли метод авторизации.
		if !requiresAuth(info.FullMethod, protectedMethods) {
			return handler(ctx, req) // Если авторизация не требуется, просто передаем запрос дальше
		}

		// Извлечение токена из метаданных запроса.
		var token string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			values := md.Get("token")
			if len(values) > 0 {
				token = values[0]
			}
		}

		if len(token) == 0 {
			// Если токен отсутствует, возвращаем ошибку Unauthorized.
			logger.Log.Info("Missing token in request", zap.String("method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}

		// Проверка токена.
		userID := auth.GetUserID(token)
		if userID == "" {
			// Если токен недействителен, возвращаем ошибку Unauthorized.
			logger.Log.Info("Invalid token", zap.String("method", info.FullMethod), zap.String("token", token))
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Добавляем userID в контекст, чтобы другие обработчики могли его использовать.
		ctx = context.WithValue(ctx, "userID", userID)

		// Пропускаем запрос дальше.
		return handler(ctx, req)
	}
}

// requiresAuth проверяет, требует ли метод авторизации.
func requiresAuth(method string, protectedMethods []string) bool {
	for _, protectedMethod := range protectedMethods {
		// Проверяем, совпадает ли метод с одним из защищённых.
		if strings.HasPrefix(method, protectedMethod) {
			return true
		}
	}
	return false
}
