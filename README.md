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
его в `oversite` только для чтения. По умолчанию используется
`${PWD}/.local/var/www/html`; для другого каталога задайте переменную одним из способов ниже.

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

Для доступа к прокси с хоста создайте игнорируемый Git файл
`compose.override.yaml` рядом с `compose.yaml`:

```yaml
services:
  overgate:
    ports:
      - "127.0.0.1:2080:2080"
```

После `make up` прокси доступен на `127.0.0.1:2080`.
Связь сервисов настроена в основном Compose через сеть `backend`: oversite
слушает `:8000`, overgate обращается к `http://oversite:8000`. Порт сайта
на хост не публикуется. Проверка сайта через прокси:

```sh
curl --noproxy '' -x http://127.0.0.1:2080 http://local.overspace/
```

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
сигнала остановки. Работает пересылка HTTP для `local.overspace` и прямого
`internet`, включая HTTPS через CONNECT и цепочку через upstream HTTP-прокси.
Для `.ygg` и `.overspace` реализована HTTP-пересылка по hex-ключу через SOCKS5
из `proxyf.yggstack`. Для приёмки на двух Docker/Yggstack-узлах выполните
`make -C tests/overlay acceptance-up` и `make -C tests/overlay acceptance-check`;
см. [описание стенда](tests/overlay/README.md).
Входящий HTTP из Yggdrasil принимает отдельный `httpin` второго overgate и
пересылает его второму oversite. Стенд проверяет эту полную цепочку.
Обычные HTTP-запросы получают `400`,
CONNECT для специальных маршрутов — `405`.
`oversite` работает как HTTP-сервер статического сайта; `OVERSITE_SITE_PATH`
позволяет изменить каталог сайта в Compose. Подробнее см. [документацию сервиса](oversite/README.md).

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
