# `overgate` — микросервис OverNode

`overgate` — HTTP forward proxy проекта OverNode. Он принимает HTTP proxy-
запросы, выбирает маршрут по цели из первой строки запроса и пересылает трафик
на локальные, интернет- или overlay-направления.

## Быстрый старт

Из корня репозитория:

```sh
make build
.local/bin/overgate
```

По умолчанию proxy слушает `:2080`. Быстрая проверка HTTP-маршрута:

```sh
curl --noproxy '' --proxy http://127.0.0.1:2080 http://example.com/
```

Для локального сайта укажите backend `host:port`:

```sh
.local/bin/overgate --proxyf.local_site=127.0.0.1:8000
curl --noproxy '' --proxy http://127.0.0.1:2080 http://local.overspace/
```

`Ctrl+C` или SIGTERM корректно останавливает сервис. Сборка контейнерного
стенда выполняется командами `make up` и `make down`.

## Структура проекта

```text
overgate/
├── cmd/overgate/          запуск процесса и загрузка конфигурации
└── internal/
    ├── app/               инициализация компонентов и lifecycle
    ├── httpin/             входящий HTTP из overlay в фиксированный backend
    └── proxyf/
        ├── routing/        разбор request-target и выбор маршрута
        ├── forwarding/     обычная HTTP-пересылка
        └── tunnel/         CONNECT-туннели
common/pkg/
├── lifecycle/             общий контракт и координация сервисов
└── network/               сетевые утилиты и overlay-резолвер
```

## Документация

- [Маршруты, протоколы и преобразования](docs/routes.md)
- [Конфигурация](docs/configuration.md)
- [HTTP-пересылка и коды ответов](docs/forwarding.md)
- [Приём HTTP из Yggdrasil (`httpin`)](docs/httpin.md)
- [HTTPS, CONNECT и цепочка прокси](docs/https.md)
- [Docker и приёмочный стенд](docs/container.md)
- [Тестовый стенд overlay](../tests/overlay/README.md)

Приёмочные сценарии и ограничения, которые ещё не реализованы, описаны в
корневой [документации проекта](../docs/proxyf-implementation-plan.md).
