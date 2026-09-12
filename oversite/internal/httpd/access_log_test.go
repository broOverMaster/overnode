package httpd

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccessLogWritesRequestOnlyToConfiguredFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	logger, closeLogger, err := newAccessLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	handler := accessLogHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, "created")
	}), logger)
	request := httptest.NewRequest(http.MethodPost, "http://example.test/assets/index.html?version=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"msg=\"HTTP request\"", "method=POST", "path=/assets/index.html", "status=201", "bytes=7", "duration="} {
		if !strings.Contains(string(contents), field) {
			t.Errorf("access record misses %q: %q", field, contents)
		}
	}
}

func TestEmptyAccessLogPathDisablesOutput(t *testing.T) {
	logger, closeLogger, err := newAccessLogger("")
	if err != nil {
		t.Fatal(err)
	}
	if logger != nil {
		t.Fatal("expected disabled access logger")
	}
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnavailableAccessLog(t *testing.T) {
	directory := t.TempDir()
	server, err := New(
		Config{ListenOn: "127.0.0.1:0", SitePath: directory, AccessLogPath: filepath.Join(directory, "missing", "access.log")},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "open HTTP access log") {
		t.Fatalf("expected access-log error, got %v", err)
	}
}
