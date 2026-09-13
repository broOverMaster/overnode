// Пакет app связывает конфигурацию и жизненный цикл overgate.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"overnode/common/pkg/config/schema"
	"overnode/common/pkg/logging"
	"overnode/gate/internal/httpin"
	"overnode/gate/internal/lifecycle"
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
	// Инициализация компонентов.
	var lifeCycles []lifecycle.LifeCycle
	_, err := proxyf.New(c.ProxyF, logger, &lifeCycles)
	if err != nil {
		return fmt.Errorf("initialize proxyf: %w", err)
	}
	_, err = httpin.New(c.HTTPIn, logger, &lifeCycles)
	if err != nil {
		return fmt.Errorf("initialize httpin: %w", err)
	}
	// Запуск компонентов.
	services := lifecycle.Start(ctx, lifeCycles)

	// Завершение одного сервиса останавливает остальные.
	return services.Wait()
}
