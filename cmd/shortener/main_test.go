package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_handleGetOriginalURL(t *testing.T) {
	type want struct {
		code        int
		location    string
		contentType string
	}
	tests := []struct {
		name          string
		inputShortURL string
		storage       *URLStorage
		want          want
	}{
		{
			name: "Test valid shortURL",
			storage: &URLStorage{URL: map[string]string{
				"abc123": "https://www.google.com",
			}},
			inputShortURL: "abc123",
			want: want{
				code:        http.StatusTemporaryRedirect,
				location:    `https://www.google.com`,
				contentType: "",
			},
		},
		{
			name: "Test not valid shortURL",
			storage: &URLStorage{URL: map[string]string{
				"abc123": "https://www.google.com",
			}},
			inputShortURL: "abc12",
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "Test empty shortURL",
			storage: &URLStorage{URL: map[string]string{
				"abc123": "https://www.google.com",
			}},
			inputShortURL: "",
			want: want{
				code: http.StatusBadRequest,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+test.inputShortURL, nil)
			w := httptest.NewRecorder()

			handler := handleGetOriginalURL(test.storage)
			handler.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, test.want.code, res.StatusCode)

			if test.want.code == http.StatusTemporaryRedirect {
				assert.Equal(t, test.want.location, res.Header.Get("Location"))
			} else {
				resBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				assert.Equal(t, "Bad request\n", string(resBody))
			}
		})
	}
}

func Test_handlePostURL(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name     string
		inputURL string
		storage  *URLStorage
		want     want
	}{
		{
			name:     "Test correct URL",
			storage:  &URLStorage{make(map[string]string)},
			inputURL: "https://www.google.com",
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
			},
		},
		{
			name:     "Test not correct URL",
			storage:  &URLStorage{make(map[string]string)},
			inputURL: "ht//www.google.com",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
			},
		},
		{
			name:     "Test empty URL",
			storage:  &URLStorage{make(map[string]string)},
			inputURL: "",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(test.inputURL))
			w := httptest.NewRecorder()

			handler := handlePostURL(test.storage)
			handler.ServeHTTP(w, req)

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
