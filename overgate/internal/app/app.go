// Пакет app связывает конфигурацию и жизненный цикл overgate.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"overnode/common/pkg/config/schema"
	crypto "overnode/common/pkg/crypto"
	"overnode/common/pkg/lifecycle"
	"overnode/common/pkg/logging"
	"overnode/common/pkg/network"
	"overnode/gate/internal/httpin"
	"overnode/gate/internal/proxyf"
)

// Config описывает конфигурацию сервиса.
type Config struct {
	Logging logging.Config `mapstructure:"logging"`
	Network network.Config `mapstructure:"network"`
	ProxyF  proxyf.Config  `mapstructure:"proxyf"`
	HTTPIn  httpin.Config  `mapstructure:"httpin"`
}

// Schema возвращает параметры сервиса.
func Schema() []schema.Field {
	fields := append(logging.Schema(), network.Schema()...)
	return append(append(fields, proxyf.Schema()...), httpin.Schema()...)
}

// Run запускает proxyf и включённый httpin; завершение одного останавливает оба.
func Run(ctx context.Context, c Config, logger *slog.Logger) error {
	// Инициализация компонентов.
	var lifeCycles []lifecycle.LifeCycle
	overlayResolver, localPublicKey, err := c.Network.NewOverlayResolver()
	if err != nil {
		return fmt.Errorf("initialize network: %w", err)
	}
	publicKey, err := crypto.PublicKeyToHex(localPublicKey)
	if err != nil {
		return fmt.Errorf("encode local public key: %w", err)
	}
	logger.Info("local node identity loaded", "public_key", publicKey)
	c.ProxyF.OverlayResolver = overlayResolver
	c.ProxyF.LocalPublicKey = localPublicKey
	_, err = proxyf.New(c.ProxyF, logger, &lifeCycles)
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
