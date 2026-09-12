# logging

Создаёт переданный в зависимости структурированный `slog.Logger`; `New` не изменяет
глобальный журнал. При использовании `Schema` вместе с `config.Load` уровнем по
умолчанию будет `info`, а форматом — `text`; `json` — явная опция. Прямой вызов
`New` требует заполнить оба поля `Config` самостоятельно.

```go
import "os"

logger, closeLogger, err := logging.New(configuration.Logging, os.Stderr)
if err != nil {
	return err
}
defer closeLogger()
logger.Info("node started", "component", "app")
```

`Schema` возвращает параметры `logging.*` для `config.Load`; каталог из `file_path`
должен существовать заранее, `New` его не создаёт. `New` дублирует записи в переданный
приёмник и, при заданном `file_path`, в файл.

Опциональный файл пишется в режиме добавления, новые файлы имеют права `0600`; ротации
нет. Закрывайте приёмник после остановки производителей записей ровно один раз: повторный вызов
`closeLogger` для файла возвращает ошибку. Обычный вызов `slog.Info/Error` не возвращает
ошибку writer. Не пишите в журнал закрытые ключи, `Authorization`, cookies или тела
запросов.
