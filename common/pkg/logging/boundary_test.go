package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestHandlerReportsWriteError(t *testing.T) {
	want := errors.New("write failed")
	logger, closeLogger, err := New(Config{Level: "INFO", Format: "JSON"}, failingWriter{want})
	if err != nil {
		t.Fatal(err)
	}
	defer closeLogger()
	if err := logger.Handler().Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)); !errors.Is(err, want) {
		t.Fatalf("expected writer error, got %v", err)
	}
}

func TestFileCloseAndWriteAfterClose(t *testing.T) {
	logger, closeLogger, err := New(Config{Level: "info", Format: "text", FilePath: filepath.Join(t.TempDir(), "app.log")}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if err := closeLogger(); err == nil {
		t.Fatal("expected repeated file close error")
	}
	if err := logger.Handler().Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)); err == nil {
		t.Fatal("expected closed file write error")
	}
}

func TestInvalidFormat(t *testing.T) {
	if _, _, err := New(Config{Level: "info", Format: "xml"}, io.Discard); err == nil {
		t.Fatal("expected format error")
	}
}
