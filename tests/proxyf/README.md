# Приёмка proxyf через Yggstack

Из корня репозитория:

```sh
make acceptance-up
make acceptance-check
```

Нужны Docker Compose с поддержкой `--wait`, Bash, curl и доступ к GitHub/реестрам
для первой сборки. Используется обычный локальный `compose.override.yaml`,
публикующий только `127.0.0.1:2080:2080`. `compose.acceptance.yaml` добавляет стенд
к основным сервисам, задаёт `proxyf.yggstack=yggstack:1080` и отключает внешний
HTTP upstream только на время приёмочного запуска. Файл основного Compose и
настройки пользователя не переписываются. Не запускайте одновременно обычный
`make up`: стенд использует те же контейнеры overgate/oversite.

Два Yggstack-узла соединены прямым TCP peer-соединением внутри `backend`.
Первый предоставляет SOCKS5; второй публикует HTTP-стенд на overlay-портах 80
и 8081 через `-remote-tcp`. HTTP-сервер работает на loopback сетевого пространства
второго узла. Внешних peers и DNS в overlay нет. TUN, host network, privileged
и NET_ADMIN не требуются; тестовые контейнеры работают под UID 65532 без capabilities.
Healthcheck источника проверяет именно HTTP через SOCKS5 и overlay, а не просто порт.

Yggstack закреплён на commit `c39db65e5bccac4cbcf712331a87ea1952cca98b`,
использующем Yggdrasil v0.5.14. Identities — публичные тестовые векторы RFC 8032,
никогда не используйте их для реальных узлов. Узел назначения:

- Ключ: `3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c`.
- IPv6: `202:15ff:41e0:bde3:b52b:6a47:aac5:9724` (проверен CLI Yggstack).

Ручная проверка:

```sh
curl --noproxy '' -x 127.0.0.1:2080 http://3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c.ygg/marker
curl --noproxy '' -x 127.0.0.1:2080 http://3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c.overspace/marker
```

Ожидается `overlay-marker`, затем исходный Host, метод, URI и тело запроса.
`make acceptance-check` проверяет оба суффикса и оба порта, POST и Host,
неверные ключи, запрет CONNECT, регрессию local/internet без внешнего интернета.
HTTP/HTTPS через upstream и TLS-сертификаты проверяются также Go-тестами этапа 4.

```sh
make acceptance-faults
```

Эта проверка временно останавливает SOCKS5, затем узел назначения. Ожидаются
ограниченные по времени `502`/`504`; local и internet продолжают работать.
Скрипт восстанавливает сервисы и повторяет проверки. Журналы сохраняются в Docker:

```sh
docker compose -f compose.yaml -f compose.override.yaml -f compose.acceptance.yaml logs --tail 50 overgate yggstack overlay-node overlay-http
```

Остановка стенда и возвращение обычной конфигурации:

```sh
make acceptance-down
make up
```

Удаляются контейнеры стенда и его сеть; образы, каталог сайта на хосте и том
остаются. Во время пользовательской приёмки стенд можно оставить запущенным.
