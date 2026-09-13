# HTTP-пересылка и маршруты

Обычные запросы к `proxyf` используют absolute-form:

```http
GET http://host/path?q=1 HTTP/1.1
```

Маршрут определяется по имени в request-target. `Host` не влияет на выбор.

| Цель | Маршрут | Результат |
| --- | --- | --- |
| `local.overspace` | `local` | `proxyf.local_site` |
| остальные `*.overspace` | `overspace` | Yggstack SOCKS5 |
| `*.ygg` | `ygg` | Yggstack SOCKS5 |
| остальные имена и IP | `internet` | напрямую или через `proxyf.http_proxy` |

`proxyf/routing` разбирает request-target, `proxyf/forwarding` выбирает
транспорт и формирует обычный HTTP-запрос для backend. Метод, path/query, тело,
сквозные заголовки и трейлеры сохраняются. Hop-by-hop-заголовки и proxy
credentials удаляются. Redirect возвращается клиенту без автоматического
перехода; Upgrade не поддерживается.

Для `local` сетевой адрес меняется на `proxyf.local_site`, а логический заголовок
становится `Host: local.overspace`:

```text
вход:  GET http://local.overspace/docs HTTP/1.1
выход: GET /docs HTTP/1.1
       Host: local.overspace
```

Для `internet` без upstream соединение устанавливается непосредственно с целью.
При заданном `proxyf.http_proxy` `net/http` отправляет upstream абсолютный
proxy-запрос; прямого fallback нет. Переменные `HTTP_PROXY` и `HTTPS_PROXY` на
транспорт сервиса не влияют.

Ошибки соединения и backend возвращают `502`, таймауты — `504`, незаданный
локальный или overlay backend — `503`, неверная proxy-форма либо ключ — `400`.
Подробные схемы сетевых границ приведены в [routes.md](routes.md).
