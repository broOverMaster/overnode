package httpd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func newAccessLogger(path string) (*slog.Logger, func() error, error) {
	if strings.TrimSpace(path) == "" {
		return nil, func() error { return nil }, nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open HTTP access log %q: %w", path, err)
	}
	return slog.New(slog.NewTextHandler(file, nil)), file.Close, nil
}

func accessLogHandler(next http.Handler, logger *slog.Logger) http.Handler {
	if logger == nil {
		return next
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		recorder := &responseRecorder{ResponseWriter: writer, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(recorder, request)
		logger.Info("HTTP request",
			"method", request.Method,
			"path", request.URL.Path,
			"status", recorder.status,
			"bytes", recorder.bytes,
			"duration", time.Since(started),
		)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
	wrote  bool
}

func (recorder *responseRecorder) WriteHeader(status int) {
	if recorder.wrote {
		return
	}
	recorder.status = status
	recorder.wrote = true
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *responseRecorder) Write(data []byte) (int, error) {
	if !recorder.wrote {
		recorder.WriteHeader(http.StatusOK)
	}
	number, err := recorder.ResponseWriter.Write(data)
	recorder.bytes += int64(number)
	return number, err
}
