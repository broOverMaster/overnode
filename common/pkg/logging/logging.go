package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// New создаёт структурированный журнал и возвращает функцию закрытия открытых ресурсов.
func New(configuration Config, console io.Writer) (*slog.Logger, func() error, error) {
	if console == nil {
		return nil, nil, fmt.Errorf("console writer must not be nil")
	}
	if err := configuration.Validate(); err != nil {
		return nil, nil, err
	}

	level := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}[strings.ToLower(configuration.Level)]

	writer := console
	closeLogger := func() error { return nil }
	if configuration.FilePath != "" {
		file, err := os.OpenFile(configuration.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file %q: %w", configuration.FilePath, err)
		}
		writer = io.MultiWriter(console, file)
		closeLogger = file.Close
	}

	options := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(configuration.Format, "json") {
		handler = slog.NewJSONHandler(writer, options)
	} else {
		handler = slog.NewTextHandler(writer, options)
	}
	return slog.New(handler), closeLogger, nil
}
