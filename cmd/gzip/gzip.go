// Модуль gzip для компрессии и декомпрессии данных.
// Позволяет отправлять данные, получать и считывать их в сжатом виде
// с использованием алгоритма сжатия gzip.
package gzip

import (
	"compress/gzip"
	"io"
	"net/http"
)

// CompressWriter предоставляет обертку для http.ResponseWriter,
// которая позволяет записывать данные в сжатом формате с использованием gzip.
type CompressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// NewCompressWriter создает новый CompressWriter, оборачивая http.ResponseWriter.
// После создания можно использовать этот объект для отправки сжатых данных.
func NewCompressWriter(w http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header возвращает заголовки ответа, позволяя управлять ими через CompressWriter.
func (c *CompressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает данные в сжатом виде в ResponseWriter.
// Эти данные будут сжаты с использованием gzip.
func (c *CompressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader отправляет статус код ответа и добавляет заголовок Content-Encoding,
// если код состояния меньше 300, указывая на то, что содержимое сжато.
func (c *CompressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Close завершает работу с gzip.Writer и отправляет данные в ResponseWriter.
func (c *CompressWriter) Close() error {
	return c.zw.Close()
}

// CompressReader предоставляет обертку для io.ReadCloser, которая позволяет
// читать данные в сжатом виде и декомпрессировать их с использованием gzip.
type CompressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// NewCompressReader создает новый CompressReader, который читает данные,
// сжатые в формате gzip, и предоставляет их в распакованном виде.
func NewCompressReader(r io.ReadCloser) (*CompressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read читает данные из сжатого потока и распаковывает их.
func (c CompressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close закрывает как исходный Reader, так и gzip.Reader.
func (c *CompressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}
