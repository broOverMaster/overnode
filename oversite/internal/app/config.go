// Пакет app связывает конфигурацию и жизненный цикл компонентов oversite.
package app

import (
	"overnode/common/pkg/config/schema"
	"overnode/common/pkg/logging"
	"overnode/site/internal/httpd"
)

// Config описывает полную конфигурацию сервиса.
type Config struct {
	Logging logging.Config `mapstructure:"logging"`
	HTTPD   httpd.Config   `mapstructure:"httpd"`
}

// Schema возвращает описание внешних параметров конфигурации сервиса.
func Schema() []schema.Field {
	return append(logging.Schema(), httpd.Schema()...)
}
