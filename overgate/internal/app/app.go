// Пакет app связывает конфигурацию и жизненный цикл overgate.
package app

import (
	"context"
	"errors"
	"log/slog"
	"overnode/common/pkg/config/schema"
	"overnode/common/pkg/logging"
	"overnode/gate/internal/httpin"
	"overnode/gate/internal/proxyf"
)

// Config описывает конфигурацию сервиса.
type Config struct {
	Logging logging.Config `mapstructure:"logging"`
	ProxyF  proxyf.Config  `mapstructure:"proxyf"`
	HTTPIn  httpin.Config  `mapstructure:"httpin"`
}

// Schema возвращает параметры сервиса.
func Schema() []schema.Field {
	return append(append(logging.Schema(), proxyf.Schema()...), httpin.Schema()...)
}

// Run запускает proxyf и включённый httpin; завершение одного останавливает оба.
func Run(ctx context.Context, c Config, logger *slog.Logger) error {
	server, err := proxyf.New(c.ProxyF, logger)
	if err != nil {
		return err
	}
	if err := c.HTTPIn.Validate(); err != nil {
		return err
	}
	if c.HTTPIn.ListenOn == "" {
		return server.Run(ctx)
	}
	incoming, err := httpin.New(c.HTTPIn, logger)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan error, 2)
	go func() { results <- server.Run(ctx) }()
	go func() { results <- incoming.Run(ctx) }()
	first := <-results
	cancel()
	return errors.Join(first, <-results)
}
