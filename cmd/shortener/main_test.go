package main

import (
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRequest(t *testing.T, ts *httptest.Server, method,
	path string) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func Test_handlePost(t *testing.T) {
	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name     string
		inputURL string
		want     want
	}{
		{
			name:     "Test correct URL",
			inputURL: "https://www.google.com",
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name:     "Test empty URL",
			inputURL: "",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.inputURL))
			w := httptest.NewRecorder()

			handlePost(w, req)

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
	r := chi.NewRouter()
	r.Get("/{shortURL}", handleGet)
	ts := httptest.NewServer(r)
	defer ts.Close()

	mu.Lock()
	urlStore["abc123"] = "https://www.google.com"
	mu.Unlock()

	type want struct {
		code     int
		location string
	}
	tests := []struct {
		name         string
		inputShortID string
		want         want
	}{
		{
			name:         "Test valid short URL",
			inputShortID: "abc123",
			want:         want{code: http.StatusOK},
		},
		{
			name:         "Test invalid short URL",
			inputShortID: "ab23",
			want:         want{code: http.StatusBadRequest},
		},
		{
			name:         "Test empty short URL",
			inputShortID: "",
			want:         want{code: http.StatusNotFound},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, _ := testRequest(t, ts, "GET", "/"+test.inputShortID)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}
