# Маршруты и протоколы overgate

Этот документ показывает путь запроса через процессы и модули OverNode. В схемах
`→` означает вызов Go-кода внутри процесса, а `=(протокол)=>` — сетевое соединение
между процессами.

## Общая точка входа

Клиент подключается к `proxyf.listen_on` (по умолчанию `:2080`) как к HTTP forward
proxy. Для обычного HTTP первая строка содержит absolute-form:

```http
GET http://example.org/path?q=1 HTTP/1.1
```

Для туннеля `internet` используется authority-form:

```http
CONNECT example.org:443 HTTP/1.1
```

Внутри overgate запрос проходит через:

```text
proxyf.Server.ServeHTTP
  → proxyf/routing.Parse
      → internal/network.ParseAuthority
  → обработчик выбранного маршрута
```

`routing` берёт цель только из первой строки запроса. Заголовок `Host` не меняет
маршрут. Обычный origin-form `GET /path HTTP/1.1` на слушателе `proxyf` получает
`400 Bad Request`. Порт обычного HTTP по умолчанию — `80`; у CONNECT порт
обязателен.

| Целевое имя | Маршрут | Обычный HTTP | CONNECT |
| --- | --- | --- | --- |
| `local.overspace` | `local` | разрешён | `405` |
| `<hex_public_key>.overspace` | `overspace` | разрешён | `405` |
| `<hex_public_key>.ygg` | `ygg` | разрешён | `405` |
| остальные имена и IP | `internet` | разрешён | разрешён |

## Маршрут local

```text
Клиент
  =(HTTP Proxy / TCP; absolute-form)=> proxyf
  → routing [local]
  → forwarding
  =(HTTP / TCP; origin-form)=> proxyf.local_site
  → oversite/httpd
```

Пример преобразования:

```text
вход:  GET http://local.overspace/docs?a=1 HTTP/1.1
выход: GET /docs?a=1 HTTP/1.1
       Host: local.overspace
```

`forwarding` меняет сетевой адрес назначения на `proxyf.local_site`, например
`oversite:8000`. Метод, path, query, тело, сквозные заголовки и трейлеры
сохраняются. Логический `Host` устанавливается в `local.overspace`.

Пустой `proxyf.local_site` возвращает `503`; ошибка соединения — `502`; таймаут —
`504`. Настройка `proxyf.http_proxy` на этот маршрут не влияет.

## Маршрут internet: обычный HTTP

### Прямое соединение

```text
Клиент
  =(HTTP Proxy / TCP; absolute-form)=> proxyf
  → routing [internet]
  → forwarding → internetTransport
  =(HTTP / TCP; origin-form)=> целевой HTTP-сервер
```

```text
вход:  GET http://example.org/path HTTP/1.1
выход: GET /path HTTP/1.1
       Host: example.org
```

Если `proxyf.http_proxy` пуст, `internetTransport` соединяется с целью напрямую.
Системные `HTTP_PROXY` и `HTTPS_PROXY` не используются.

### Через upstream HTTP proxy

```text
Клиент
  =(HTTP Proxy / TCP; absolute-form)=> proxyf
  → routing [internet]
  → forwarding → internetTransport
  =(HTTP Proxy / TCP; absolute-form)=> proxyf.http_proxy
  =(HTTP / TCP; обычно origin-form)=> целевой HTTP-сервер
```

Для upstream запрос остаётся proxy-запросом с абсолютным URL. Последующее
преобразование выполняет upstream proxy и не контролируется overgate. При отказе
upstream прямого fallback нет.

## Маршрут internet: CONNECT и HTTPS

CONNECT обрабатывает `proxyf/tunnel`, а не `forwarding`.

### Прямой туннель

```text
Клиент
  =(HTTP CONNECT / TCP)=> proxyf
  → routing [internet]
  → tunnel.Connect
  =(TCP)=> целевой сервер:443
  <=(200 Connection Established)= proxyf
Клиент =(TLS внутри TCP-туннеля)=> целевой HTTPS-сервер
```

После ответа `200` overgate копирует байты в обе стороны и не разбирает TLS,
HTTP внутри TLS, сертификаты или имя SNI. Поддерживается TCP half-close.

### Туннель через upstream HTTP proxy

```text
Клиент
  =(HTTP CONNECT / TCP)=> proxyf
  → routing [internet]
  → tunnel.Connect
  =(TCP)=> proxyf.http_proxy
  =(HTTP CONNECT по этому TCP-соединению)=> upstream proxy
  =(TCP)=> целевой сервер:443
  <=(200 Connection Established)= proxyf
Клиент =(TLS внутри составного TCP-туннеля)=> целевой HTTPS-сервер
```

Overgate отправляет клиенту `200` только после прямого соединения или успешного
ответа 2xx от upstream. Ошибка соединения либо отказ upstream даёт `502`, таймаут
установления TCP/CONNECT — `504`. Активные соединения учитывает `tunnel.Tracker`;
остановка proxyf закрывает обе стороны туннеля.

HTTP CONNECT и SOCKS5 CONNECT — разные протоколы. Первый приходит от HTTP-клиента
и разрешён только для `internet`. Второй используется далее маршрутами `ygg` и
`overspace` как команда Yggstack открыть TCP-соединение.

## Маршрут ygg

```text
Клиент
  =(HTTP Proxy / TCP; absolute-form)=> overgate A / proxyf
  → routing [ygg]
  → common/pkg/overlay.NormalizeAddress
      <hex_public_key>.ygg → IPv6 Yggdrasil
  → forwarding → overlayTransport
  =(SOCKS5 / TCP; CONNECT к [IPv6]:port)=> Yggstack A
  =(HTTP / TCP / IPv6 поверх Yggdrasil)=> Yggdrasil-узел назначения
```

После SOCKS5 handshake по установленному TCP-потоку передаётся обычный HTTP:

```text
вход proxyf:       GET http://<key>.ygg/path HTTP/1.1
через Yggdrasil:   GET /path HTTP/1.1
                    Host: <key>.ygg
```

`common/pkg/overlay` вычисляет IPv6 из 32-байтного публичного ключа Ed25519. Это
локальное вычисление, DNS-запроса нет. В SOCKS5 передаётся числовой IPv6 и порт
из URL, по умолчанию `80`. `forwarding` сохраняет исходный логический `Host`.

Принимающий Yggdrasil-узел должен предоставить TCP-сервис на целевом порту. Это
может быть HTTP-сервер в Yggdrasil либо реализованная в стенде связка
`Yggstack -remote-tcp → overgate/httpin`.

## Маршрут overspace

Сейчас `overspace` и `ygg` имеют отдельные значения маршрута, но используют
одинаковое преобразование ключа, `overlayTransport` и SOCKS5 Yggstack. Такое
разделение сохраняет место для дальнейшей логики overspace. Резолвинг имени,
отличного от `<hex_public_key>.overspace`, отложен в задачу T-2.

Полная реализованная схема приёмочного стенда:

```text
Клиент
  =(HTTP Proxy / TCP; absolute-form)=> overgate A
  → proxyf.Server
  → routing [overspace]
  → common/pkg/overlay: public key → IPv6
  → forwarding → overlayTransport
  =(SOCKS5 / TCP; CONNECT к [IPv6]:80)=> Yggstack A
  =(HTTP / TCP / IPv6; шифрование Yggdrasil)=> Yggstack B
  → Yggstack B -remote-tcp
  =(HTTP / локальный TCP; origin-form)=> 127.0.0.1:8080
  → overgate B / httpin
  → httputil.ReverseProxy
  =(HTTP / TCP; origin-form)=> httpin.local_site
  → oversite B / httpd
```

В стенде `Yggstack B` запускается с отображениями
`80:127.0.0.1:8080` и `8081:127.0.0.1:8080`. `overgate B` разделяет с ним
сетевое пространство контейнера и слушает `httpin` на `127.0.0.1:8080`.
Backend задаётся как `overlay-site:8000`.

Преобразование HTTP в этой цепочке:

```text
клиент → proxyf A:
  GET http://<key>.overspace/marker HTTP/1.1

forwarding A → Yggstack A → Yggdrasil → remote-tcp → httpin B:
  GET /marker HTTP/1.1
  Host: <key>.overspace

httpin B → oversite B:
  GET /marker HTTP/1.1
  Host: <key>.overspace
```

SOCKS5 используется только между `overlayTransport` и исходящим Yggstack A.
`remote-tcp` на принимающей стороне работает с TCP-потоком и не знает о SOCKS5.
HTTP не преобразуется в иной прикладной протокол. Шифрование между Yggdrasil-
узлами предоставляет Yggdrasil; overgate не добавляет TLS к HTTP.

Для обоих overlay-маршрутов неверный ключ возвращает `400`, пустой
`proxyf.yggstack` — `503`, ошибка SOCKS5 или недоступность цели — `502`, таймаут —
`504`. Прямого fallback в DNS или internet нет.

## Входящий компонент httpin

`httpin` не является HTTP forward proxy. Он принимает обычный origin-form HTTP:

```http
GET /path?q=1 HTTP/1.1
Host: <key>.overspace
```

```text
Yggstack -remote-tcp или другой доверенный вход
  =(HTTP / TCP; origin-form)=> httpin.listen_on
  → httpin.Server
  → httputil.ReverseProxy
  =(HTTP / TCP; origin-form)=> httpin.local_site
```

`httpin` всегда выбирает фиксированный `httpin.local_site`. Заголовок `Host` и
URL не запускают `proxyf/routing` и не выбирают backend. Метод, path, query,
тело, сквозные заголовки и трейлеры сохраняются. Absolute-form получает `400`,
CONNECT — `405`, пустой backend — `503`, ошибка backend — `502`, таймаут — `504`.

## Роли модулей

| Модуль | Роль | Сетевой протокол |
| --- | --- | --- |
| `internal/app` | Инициализирует компоненты и передаёт их lifecycle общему координатору. | Не участвует в трафике. |
| `common/pkg/lifecycle` | Запускает сервисы, отменяет соседние при завершении одного и собирает ошибки. | Не участвует в трафике. |
| `internal/proxyf` | HTTP forward proxy, политика доступности маршрутов и владелец транспортов. | Принимает HTTP Proxy и CONNECT. |
| `internal/network` | Проверяет и разбирает `host[:port]`/`host:port`. | Сетевых обращений нет. |
| `internal/proxyf/routing` | Разбирает request-target, нормализует имя и выбирает маршрут. | Сетевых обращений нет. |
| `common/pkg/overlay` | Преобразует публичный ключ Ed25519 в IPv6 Yggdrasil. | Сетевых обращений и DNS нет. |
| `internal/proxyf/forwarding` | Формирует исходящий HTTP-запрос и выбирает готовый transport маршрута. | HTTP через выбранный transport. |
| `internal/proxyf/tunnel` | Обслуживает прямые и chained CONNECT-туннели и закрывает активные соединения. | HTTP CONNECT, затем прозрачный TCP. |
| `internal/httpin` | Принимает origin-form HTTP и пересылает фиксированному backend. | HTTP server и HTTP client. |
| `oversite/internal/httpd` | Обслуживает локальный сайт. | HTTP server. |

## Где меняются адрес и протокол

| Граница | Вход | Выход | Изменение |
| --- | --- | --- | --- |
| `routing` | HTTP request-target | `routing.Target` | Текст разбирается в route, host, port и URL. |
| `overlay` | `<key>.ygg` или `<key>.overspace` | IPv6 | Детерминированное вычисление адреса. |
| `forwarding` | absolute-form HTTP | origin-form HTTP | Меняется URL назначения; Host сохраняется, кроме `local`. |
| `overlayTransport` | Dial TCP к IPv6 | SOCKS5 CONNECT | Yggstack создаёт TCP-поток через Yggdrasil. |
| Yggdrasil | TCP/IPv6 | зашифрованный overlay-трафик | Шифрование и доставка выполняются Yggdrasil. |
| `remote-tcp` | Входящий Yggdrasil TCP | локальный TCP | Содержимое потока не переписывается. |
| `httpin` | origin-form HTTP | origin-form HTTP к backend | Меняется сетевой backend; Host и URL запроса сохраняются. |
| `tunnel` | HTTP CONNECT | прозрачный TCP | После 200 HTTP-разбор заканчивается; TLS остаётся end-to-end. |

## Коды ошибок

| Код | Основные причины |
| --- | --- |
| `400` | Неверная форма proxy-запроса, target, имя или публичный ключ; absolute-form на `httpin`. |
| `405` | CONNECT для `local`, `ygg`, `overspace` либо любой CONNECT на `httpin`. |
| `502` | Ошибка HTTP backend, upstream proxy, SOCKS5 или установки TCP-соединения. |
| `503` | Не настроены `proxyf.local_site`, `proxyf.yggstack` или backend `httpin`. |
| `504` | Таймаут соединения, SOCKS5/CONNECT handshake либо ожидания заголовков ответа. |

Служебные hop-by-hop заголовки удаляются при HTTP-пересылке. Redirect возвращается
клиенту без автоматического перехода; HTTP Upgrade не поддерживается.
