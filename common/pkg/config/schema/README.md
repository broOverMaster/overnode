# config/schema

Описывает внешние поля конфигурации для `config.Load`. Используйте конструкторы
`String`, `Uint16`, `Int`, `Duration` и `StringSlice`: они связывают имя, тип,
значение по умолчанию и текст справки флага.

```go
import (
	"time"

	"overnode/common/pkg/config/schema"
)

func Schema() []schema.Field {
	return []schema.Field{
		schema.String("httpd.listen_on", "127.0.0.1:8000", "HTTP listen address"),
		schema.Duration("httpd.shutdown_timeout", 30*time.Second, "graceful shutdown timeout"),
	}
}
```

Имя состоит из строчных идентификаторов, разделённых точками. `Combine` объединяет
схемы компонентов и отвергает повторы, отношение «родитель—потомок» и имена, которые
после замены точек на `__` совпали бы в переменных окружения. Все значения по
умолчанию должны иметь тип выбранного конструктора.

Общий загрузчик допускает, например, порт `0`; конкретный сервис обязан отклонить
недопустимые для него значения в собственной `Validate`.
