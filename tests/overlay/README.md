# Приёмка overlay-сети OverNode

Стенд запускается из этого каталога или из корня репозитория:

```sh
cd tests/overlay
make acceptance-up
make acceptance-check
```

Нужны Docker Compose с поддержкой `--wait`, Bash, curl и доступ к GitHub/реестрам
для первой сборки. По умолчанию proxy узла `broOverMaster` публикуется на
`127.0.0.1:2080`; для одновременной работы с основной нодой используйте другой
порт через `ACCEPTANCE_PORT`.

Все три `overgate` используют общий пример конфигурации
`.local/etc/overgate/config.yaml`, а ключ узла выбирается переменной
`NETWORK__LOCAL_KEY_PATH`:

- `broOverMaster` — `.local/etc/local-key.pem`;
- `alice` — `.local/etc/alice-key.pem`;
- `bob` — `.local/etc/bob-key.pem`.

Каждый узел имеет собственный `yggstack`, `overgate` и `oversite`. Все три
`yggstack` подключены к отдельной чистой ноде Yggdrasil, которая также имеет
пиры на каждом из них. Через proxy `broOverMaster` проверяются HTTP-запросы к
`broOverMaster.overspace`, `alice.overspace` и `bob.overspace`; разные ответы
`served-by` подтверждают доставку запроса до нужного узла.

Конфигурации Yggstack и Yggdrasil находятся в `.local/etc/yggstack` и
`.local/etc/yggdrasil`. Чистая нода запускается с TUN-интерфейсом.

Проверка отказа:

```sh
make acceptance-faults
```

Остановка стенда:

```sh
make acceptance-down
```
