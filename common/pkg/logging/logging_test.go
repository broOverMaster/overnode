package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConsoleTextLoggingAndLevelFiltering(t *testing.T) {
	var console bytes.Buffer
	logger, closeLogger, err := New(Config{Level: "info", Format: "text"}, &console)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closeLogger(); err != nil {
			t.Error(err)
		}
	})

	logger.Debug("hidden")
	logger.Info("node started", slog.String("component", "app"))
	if strings.Contains(console.String(), "hidden") {
		t.Fatalf("filtered record was written: %q", console.String())
	}
	if !strings.Contains(console.String(), `level=INFO msg="node started" component=app`) {
		t.Fatalf("unexpected text log: %q", console.String())
	}
}

func TestConsoleJSONLogging(t *testing.T) {
	var console bytes.Buffer
	logger, closeLogger, err := New(Config{Level: "debug", Format: "json"}, &console)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closeLogger(); err != nil {
			t.Error(err)
		}
	})

	logger.Warn("configuration changed", slog.String("source", "environment"))
	var record map[string]any
	if err := json.Unmarshal(console.Bytes(), &record); err != nil {
		t.Fatalf("decode JSON log: %v; record: %q", err, console.String())
	}
	if record["level"] != "WARN" || record["msg"] != "configuration changed" || record["source"] != "environment" {
		t.Fatalf("unexpected JSON log: %#v", record)
	}
}

func TestFileAppendsAndDuplicatesRecordsToConsole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overnode.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var console bytes.Buffer
	logger, closeLogger, err := New(Config{Level: "info", Format: "json", FilePath: path}, &console)
	if err != nil {
		t.Fatal(err)
	}
	logger.Error("request failed", slog.Int("status", 500))
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(contents), "existing\n") || !strings.Contains(string(contents), `"msg":"request failed"`) {
		t.Fatalf("unexpected log file: %q", contents)
	}
	if !strings.Contains(console.String(), `"msg":"request failed"`) {
		t.Fatalf("record was not duplicated to console: %q", console.String())
	}
}

func TestNewFileUsesOwnerOnlyPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overnode.log")
	logger, closeLogger, err := New(Config{Level: "info", Format: "text", FilePath: path}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("created")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("unexpected file permissions: %o", permissions)
	}
}

func TestRejectsNilConsoleAndInvalidConfiguration(t *testing.T) {
	if _, _, err := New(Config{Level: "info", Format: "text"}, nil); err == nil {
		t.Fatal("expected a nil-console error")
	}
	if _, _, err := New(Config{Level: "trace", Format: "text"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an invalid-configuration error")
	}
}

func TestRejectsFileInMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "overnode.log")
	_, _, err := New(Config{Level: "info", Format: "text", FilePath: path}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "open log file") {
		t.Fatalf("expected an open-file error, got %v", err)
	}
}
