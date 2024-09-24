package logger

import (
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var Log = zap.NewNop()

func Initialize(level string) error {

	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()

	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	Log = zl
	return nil
}

func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	logFn := func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		uri := r.RequestURI
		method := r.Method

		h(w, r)

		duration := time.Since(start)

		Log.Info("got incoming HTTP request",
			zap.String("path", uri),
			zap.String("method", method),
			zap.String("duration", strconv.FormatInt(int64(duration), 10)),
		)

	}

	return logFn
}
