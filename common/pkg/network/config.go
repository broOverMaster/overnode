package network

import (
	"fmt"

	"overnode/common/pkg/config/schema"
	crypto "overnode/common/pkg/crypto"
	"overnode/common/pkg/network/overlay"
)

// Config задаёт настройки разрешения имён overlay-сети.
type Config struct {
	LocalKey         string `mapstructure:"local_key"`
	LocalKeyPath     string `mapstructure:"local_key_path"`
	OverlayHostsPath string `mapstructure:"overlay_hosts_path"`
}

// Schema описывает обязательный ключ локальной ноды и hosts-файл.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("network.local_key", "", "required local Ed25519 key in hex"),
		schema.String("network.local_key_path", "", "optional path to local Ed25519 PEM key"),
		schema.String("network.overlay_hosts_path", "", "optional overlay hosts file path"),
	}
}

// NewOverlayResolver создаёт резолвер по конфигурации network.
func (configuration Config) NewOverlayResolver() (OverlayResolver, []byte, error) {
	var publicKey []byte
	var err error
	if configuration.LocalKey != "" {
		publicKey, err = crypto.PubFromHex(configuration.LocalKey)
	} else if configuration.LocalKeyPath != "" {
		var pemKey []byte
		pemKey, err = crypto.PubFromPEMFile(configuration.LocalKeyPath)
		publicKey = pemKey
	} else {
		err = fmt.Errorf("one of network.local_key or network.local_key_path is required")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("local key: %w", err)
	}
	resolver, err := overlay.NewResolver(publicKey, configuration.OverlayHostsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("network overlay resolver: %w", err)
	}
	return resolver, append([]byte(nil), publicKey...), nil
}
