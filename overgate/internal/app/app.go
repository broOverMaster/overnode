// Пакет app связывает конфигурацию и жизненный цикл overgate.
package app

import (
	"context"
	"log/slog"
	"overnode/common/pkg/config/schema"
	"overnode/common/pkg/logging"
	"overnode/gate/internal/proxyf"
)

// Config описывает конфигурацию сервиса.
type Config struct {
	Logging logging.Config `mapstructure:"logging"`
	ProxyF  proxyf.Config  `mapstructure:"proxyf"`
}

// Schema возвращает параметры сервиса.
func Schema() []schema.Field { return append(logging.Schema(), proxyf.Schema()...) }

// Run создаёт прокси и обслуживает запросы до отмены контекста.
func Run(ctx context.Context, c proxyf.Config, logger *slog.Logger) error {
	server, err := proxyf.New(c, logger)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}
