# Приёмка proxyf через Yggstack

Команды стенда определены только в `tests/proxyf/Makefile`; корневой Makefile
его не подключает. Перейдите в `tests/proxyf`: обычный `make` запускает стенд.
Из корня можно явно вызвать `make -C tests/proxyf acceptance-up`.

```sh
cd tests/proxyf
make acceptance-up
make acceptance-check
```

Нужны Docker Compose с поддержкой `--wait`, Bash, curl и доступ к GitHub/реестрам
для первой сборки. Самостоятельный `tests/proxyf/compose.yaml` создаёт проект
`overnode-acceptance`, не подключая корневые Compose-файлы и настройки сервисов рабочей ноды.
Все данные сайта и identities находятся здесь же. По умолчанию публикуется
`127.0.0.1:2080:2080`; для одновременной работы с основной нодой выберите другой порт:

```sh
ACCEPTANCE_PORT=22080 make acceptance-up
ACCEPTANCE_PORT=22080 make acceptance-check
ACCEPTANCE_PORT=22080 make acceptance-faults
```

Стенд задаёт `proxyf.yggstack=yggstack:1080`, использует локальный Ed25519-ключ
из `tests/proxyf/local-key.pem` через `network.local_key_path` и не использует
внешний HTTP upstream.

Два Yggstack-узла соединены прямым TCP peer-соединением внутри `backend`.
Первый предоставляет SOCKS5; второй публикует входящий HTTP-компонент второго
overgate (`overlay-gate`) на overlay-портах 80 и 8081 через `-remote-tcp`.
`httpin` слушает `127.0.0.1:8080` в сетевом пространстве второго Yggstack и
пересылает запросы настоящему второму oversite (`overlay-site:8000`).
Этот сайт подключён только к отдельной внутренней сети `destination`:
первый overgate не имеет прямого доступа к нему. Внешних peers и DNS в overlay нет. TUN, host network, privileged
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

Ожидается `overlay-marker` и `served-by=oversite-B`. Полная цепочка:

```text
overgate A / proxyf → Yggstack A / SOCKS5 → Yggdrasil
  → Yggstack B / remote-tcp → overgate B / httpin → oversite B
```

`make acceptance-check` проверяет оба суффикса и оба порта, GET-маркер,
ответ `200` от настоящего статического oversite на POST,
неверные ключи, запрет CONNECT, регрессию local/internet без внешнего интернета.
HTTP/HTTPS через upstream и TLS-сертификаты проверяются также Go-тестами этапа 4.

```sh
make acceptance-faults
```

Эта проверка временно останавливает SOCKS5, узел назначения, второй overgate
и второй oversite. Ожидаются
ограниченные по времени `502`/`504`; local и internet продолжают работать.
Скрипт восстанавливает сервисы и повторяет проверки. Журналы сохраняются в Docker:

```sh
docker compose -f compose.yaml logs --tail 50 overgate yggstack overlay-node overlay-gate overlay-site
```

Остановка стенда:

```sh
make acceptance-down
```

Удаляются только контейнеры и сети проекта `overnode-acceptance`; образы и файлы
сайта остаются. Рабочая нода запускается независимо через `make up` из корня.
