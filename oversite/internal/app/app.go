package app

import (
	"context"
	"log/slog"

	"overnode/site/internal/httpd"
)

// Run создаёт HTTP-компонент и обслуживает сайт до отмены контекста.
func Run(ctx context.Context, configuration httpd.Config, logger *slog.Logger) error {
	server, err := httpd.New(configuration, logger)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}
