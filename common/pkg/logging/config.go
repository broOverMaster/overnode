// Пакет logging создаёт и настраивает журнал приложения.
package logging

import (
	"fmt"
	"strings"

	"overnode/common/pkg/config/schema"
)

// Config содержит настройки фильтрации, формата и назначения журнала.
type Config struct {
	Level    string `mapstructure:"level" json:"level" yaml:"level" toml:"level"`
	Format   string `mapstructure:"format" json:"format" yaml:"format" toml:"format"`
	FilePath string `mapstructure:"file_path" json:"file_path" yaml:"file_path" toml:"file_path"`
}

// Schema возвращает внешние параметры журналирования.
func Schema() []schema.Field {
	return []schema.Field{
		schema.String("logging.level", "info", "log level: debug, info, warn or error"),
		schema.String("logging.format", "text", "log format: text or json"),
		schema.String("logging.file_path", "", "optional log file path; logs are also written to stderr"),
	}
}

// Validate проверяет поддерживаемые уровень и формат журнала.
func (configuration Config) Validate() error {
	switch strings.ToLower(configuration.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be one of debug, info, warn or error")
	}

	switch strings.ToLower(configuration.Format) {
	case "text", "json":
	default:
		return fmt.Errorf("logging.format must be text or json")
	}
	return nil
}
