package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/sol1corejz/go-url-shortener/cmd/config"
	"github.com/sol1corejz/go-url-shortener/internal/handlers"
	"github.com/sol1corejz/go-url-shortener/internal/middlewares"
	"github.com/sol1corejz/go-url-shortener/internal/models"
	"github.com/sol1corejz/go-url-shortener/internal/storage"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initStorageForTests(t *testing.T, storageType string) storage.Storage {
	var store storage.Storage
	var err error

	switch storageType {
	case "postgres":
		dsn := "your_postgres_dsn"
		store, err = storage.NewPostgresStorage(dsn)
		require.NoError(t, err)
	case "file":
		tmpFile, err := os.CreateTemp("", "test_file_*.json")
		require.NoError(t, err)
		t.Cleanup(func() { os.Remove(tmpFile.Name()) })
		store, err = storage.NewFileStorage(tmpFile.Name())
		require.NoError(t, err)
	case "memory":
		store = storage.NewMemoryStorage()
	default:
		t.Fatalf("Invalid storage type: %s", storageType)
	}

	return store
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func initFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_file_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	config.FileStoragePath = tmpFile.Name()
}

func Test_handlePost(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name        string
		inputURL    string
		storageType string
		want        want
	}{
		{
			name:        "Test correct URL with memory storage",
			inputURL:    "https://www.google.com",
			storageType: "memory",
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:        "Test empty URL with file storage",
			inputURL:    "",
			storageType: "file",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := initStorageForTests(t, test.storageType)
			handler := handlers.NewHandler(store)

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.inputURL))
			w := httptest.NewRecorder()

			handler.HandlePost(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, test.want.code, res.StatusCode)

			if test.want.code == http.StatusCreated {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))

				shortURLs := strings.Split(string(resBody), "/")
				shortURL := shortURLs[len(shortURLs)-1]
				assert.Len(t, shortURL, 8)
			}
		})
	}
}

func Test_handleGet(t *testing.T) {
	type want struct {
		code int
	}
	tests := []struct {
		name         string
		inputShortID string
		storageType  string
		want         want
	}{
		{
			name:         "Test valid short URL with memory storage",
			inputShortID: "abc123",
			storageType:  "memory",
			want:         want{code: http.StatusOK},
		},
		{
			name:         "Test invalid short URL with file storage",
			inputShortID: "ab23",
			storageType:  "file",
			want:         want{code: http.StatusNotFound},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := initStorageForTests(t, test.storageType)

			if test.storageType != "file" {
				store.Save(models.URLData{
					ShortURL:    "abc123",
					OriginalURL: "https://www.google.com",
					UUID:        "test-uuid",
				})
			}

			handler := handlers.NewHandler(store)

			r := chi.NewRouter()
			r.Get("/{shortURL}", handler.HandleGet)

			ts := httptest.NewServer(r)
			defer ts.Close()

			resp, _ := testRequest(t, ts, "GET", "/"+test.inputShortID)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

func Test_handleJSONPost(t *testing.T) {
	type want struct {
		code        int
		contentType string
		result      models.Response
	}
	tests := []struct {
		name        string
		reqBody     models.Request
		storageType string
		want        want
	}{
		{
			name: "Valid request with memory storage",
			reqBody: models.Request{
				URL: "https://practicum.yandex.ru",
			},
			storageType: "memory",
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				result:      models.Response{},
			},
		},
		{
			name: "Invalid JSON with file storage",
			reqBody: models.Request{
				URL: "",
			},
			storageType: "file",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				result:      models.Response{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := initStorageForTests(t, test.storageType)
			handler := handlers.NewHandler(store)

			r := chi.NewRouter()
			r.Post("/api/shorten", handler.HandleJSONPost)

			ts := httptest.NewServer(r)
			defer ts.Close()

			reqBodyJSON, _ := json.Marshal(test.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBodyJSON))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, test.want.code, rr.Code)
			assert.Equal(t, test.want.contentType, rr.Header().Get("Content-Type"))

			if test.want.code == http.StatusCreated {
				var resp models.Response
				err := json.Unmarshal(rr.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.Result)
			} else {
				assert.Equal(t, rr.Body.String(), "Empty URL\n")
			}
		})
	}
}

func TestGzipCompression(t *testing.T) {
	store := initStorageForTests(t, "memory")
	handler := handlers.NewHandler(store)

	config.FlagBaseURL = "http://localhost:8080"
	r := chi.NewRouter()
	r.Post("/api/shorten", middlewares.GzipMiddleware(handler.HandleJSONPost))

	ts := httptest.NewServer(r)
	defer ts.Close()

	requestBody := `{"url": "https://yypo2q5oco9.net"}`

	t.Run("sends_gzip", func(t *testing.T) {
		buf := bytes.NewBuffer(nil)
		zb := gzip.NewWriter(buf)

		_, err := zb.Write([]byte(requestBody))
		require.NoError(t, err)

		err = zb.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten", buf)
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "")

		rr := httptest.NewRecorder()

		handler := middlewares.GzipMiddleware(handler.HandleJSONPost)
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusCreated {
			var resp models.Response
			err := json.Unmarshal(rr.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.NotEmpty(t, resp.Result)
		} else {
			assert.Equal(t, rr.Body.String(), "Empty URL\n")
		}
	})

	t.Run("accepts_gzip", func(t *testing.T) {
		buf := bytes.NewBufferString(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten", buf)
		req.Header.Set("Accept-Encoding", "gzip")

		rr := httptest.NewRecorder()

		handler := middlewares.GzipMiddleware(handler.HandleJSONPost)
		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusCreated {
			zresp, err := gzip.NewReader(rr.Body)
			require.NoError(t, err)

			res, err := io.ReadAll(zresp)
			require.NoError(t, err)
			assert.NotEmpty(t, res)
		} else {
			assert.Equal(t, rr.Body.String(), "Empty URL\n")
		}
	})
}
