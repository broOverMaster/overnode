# OverNode

OverNode — набор Go-микросервисов. Проект находится на начальном этапе: сервисы уже собираются как отдельные контейнерные образы, а прикладная логика и внешние интерфейсы ещё формируются.

## Состав проекта

```text
.
├── common/       общие Go-пакеты для сервисов
├── overgate/     сервис-шлюз
├── oversite/     сервис локального сайта
├── compose.yaml  сборка и запуск контейнеров
├── go.work       локальный workspace Go
└── Makefile      основные команды разработки
```

`common`, `overgate` и `oversite` — самостоятельные Go-модули, объединённые в корневой `go.work`. Это позволяет сервисам использовать общие пакеты напрямую во время локальной разработки.

## Требования

- Go 1.26 или новее;
- Docker Engine с Docker Compose v2 — для контейнерного запуска;
- GNU Make — для команд из `Makefile`.

## Быстрый старт

Проверить форматирование, статический анализ, тесты и локальную сборку:

```sh
make check
```

Собрать образы и запустить сервисы:

```sh
OVERSITE_SITE_PATH="$PWD/.local/var/www/html" make up
```

`OVERSITE_SITE_PATH` — путь к каталогу статического сайта на хосте; Compose монтирует
его в `oversite` только для чтения. Его необходимо задать одним из способов ниже.

Можно передать путь при запуске:

```sh
OVERSITE_SITE_PATH="$PWD/.local/var/www/html" make up
```

Или создайте локальный файл `.env` рядом с `compose.yaml`:

```dotenv
OVERSITE_SITE_PATH=${PWD}/.local/var/www/html
```

Docker Compose читает `.env` автоматически, поэтому после этого достаточно выполнить
`docker compose up`. Для `OVERSITE_SITE_PATH` нужен абсолютный путь: bind mount создаёт
Docker daemon, а не процесс Compose в текущем каталоге. Файл `.env` уже исключён из Git.

Для постоянной локальной конфигурации и доступа к `oversite` с хоста создайте
игнорируемый Git файл `compose.override.yaml` рядом с `compose.yaml`. Compose подключит
его автоматически. Host network сохраняет HTTP-слушатель на `127.0.0.1`, а `!reset`
удаляет унаследованную сеть `backend`, несовместимую с `network_mode`:

```yaml
services:
  oversite:
    network_mode: host
    networks: !reset []

volumes:
  oversite-site:
    driver_opts:
      device: ${PWD}/.local/var/www/html
```

После этого достаточно выполнить `docker compose up`, а страницу можно открыть на
<http://127.0.0.1:8000/>. `compose.override.yaml` не задаёт `OVERSITE_SITE_PATH`:
переменные из `environment` доступны только контейнеру и не участвуют в подстановке
полей Compose. Поэтому override переопределяет `device` тома напрямую. Режим host
network рассчитан на Docker Engine в Linux; для Docker Desktop используйте запуск
сервиса непосредственно на хосте.

Быстро проверить `oversite` с примером локальной конфигурации и тестовой страницей:

```sh
go run oversite/cmd/oversite/main.go --config=.local/etc/config.yaml
```

После запуска откройте <http://127.0.0.1:8000/>. Для остановки нажмите `Ctrl+C`.

Остановить и удалить контейнеры и сеть Compose:

```sh
make down
```

`overgate` запускает слушатель HTTP-прокси на `:2080` (все интерфейсы) и работает до
сигнала остановки. Реализованы lifecycle и классификация маршрутов: допустимые
proxy-запросы пока получают `501`, обычные HTTP-запросы — `400`, CONNECT для
специальных маршрутов — `405`. Пересылка будет добавлена следующими этапами.
`oversite` работает как HTTP-сервер статического сайта и требует переменную
`OVERSITE_SITE_PATH` при запуске через Compose; подробнее см. [документацию сервиса](oversite/README.md).

## Команды разработки

| Команда | Назначение |
| --- | --- |
| `make build` | Собрать `overgate` и `oversite` в `.local/bin/`. |
| `make test` | Запустить тесты всех пакетов. |
| `make vet` | Выполнить `go vet`. |
| `make fmt` | Отформатировать Go-код. |
| `make fmt-check` | Проверить форматирование без изменений файлов. |
| `make check` | Выполнить `fmt-check`, `vet`, `test` и `build`. |
| `make up` | Собрать образы и запустить Compose-проект в фоне. |
| `make down` | Остановить Compose-проект. |

## Контейнеры

Каждый сервис собирается отдельным multi-stage Dockerfile. В build-stage создаётся минимальный временный Go-workspace:

- образ `overgate` содержит только модули `common` и `overgate`;
- образ `oversite` содержит только модули `common` и `oversite`.

Так сервисы не зависят от исходников друг друга при сборке образов. Контекст сборки обоих образов — корень репозитория, чтобы можно было подключать `common`.

Compose создаёт внутреннюю сеть `backend`. Порты наружу пока не публикуются.

## Документация компонентов

- [Общие пакеты](common/README.md)
- [`overgate`](overgate/README.md)
- [`oversite`](oversite/README.md)
