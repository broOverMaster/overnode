# Docker и приёмочный стенд

Рабочий Compose-файл находится в корне проекта. Override публикует только
`overgate` на `127.0.0.1:2080`; адреса сервисов задаются основной конфигурацией.

```sh
make up
curl --noproxy '' -i -x http://127.0.0.1:2080 http://local.overspace/
curl --noproxy '' -i -x http://127.0.0.1:2080 http://example.com/
make down
```

Для полного Yggdrasil/overspace-контура используется изолированный стенд
`tests/overlay`:

```sh
make -C tests/overlay acceptance-up
make -C tests/overlay acceptance-check
make -C tests/overlay acceptance-down
```

Стенд содержит source Yggstack с SOCKS5 `:1080`, destination Yggstack с
`-remote-tcp` на `overgate B/httpin :8080`, второй `oversite` и отдельные сети.
Публикуемый порт задаётся `ACCEPTANCE_PORT` (по умолчанию `2080`). Подробные
сценарии отказов находятся в [README стенда](../../tests/overlay/README.md).

В рабочем Compose `oversite` слушает `:8000`, `proxyf.local_site` указывает на
`oversite:8000`, а наружу публикуется только proxy-порт.

Образ overgate собирается из корня:

```sh
docker build -f overgate/Dockerfile .
```

В build-stage создаётся временный workspace для модулей `common` и `overgate`;
исходники `oversite` в образ не входят. Процесс запускается пользователем
`65532:65532`, слушает `:2080` на всех интерфейсах контейнера, а публикация
порта задаётся Compose override. Локальные YAML-файлы конфигурации в контейнер
не монтируются; параметры передаются через environment.
