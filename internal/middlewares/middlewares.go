// Package middlewares содержит промежуточные обработчики (middleware), которые
// выполняются во время обработки HTTP-запросов и отвечают за различные функциональности,
// такие как сжатие данных через Gzip.
package middlewares

import (
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
