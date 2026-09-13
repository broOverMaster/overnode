# HTTPS, CONNECT и цепочка прокси

CONNECT разрешён только для маршрута `internet` и обрабатывается модулем
`proxyf/tunnel`.

При прямом соединении:

```text
Клиент =(HTTP CONNECT / TCP)=> overgate
       → tunnel.Connect
       =(TCP)=> target:443
       <= 200 Connection Established
Клиент =(TLS внутри прозрачного TCP-туннеля)=> target
```

При заданном `proxyf.http_proxy` overgate сначала устанавливает TCP к upstream,
отправляет ему HTTP CONNECT к исходной цели и только после ответа 2xx сообщает
клиенту `200`. TLS остаётся между клиентом и конечным сервером; overgate не
расшифровывает TLS и не видит HTTP внутри него.

`tunnel` поддерживает half-close и отслеживает активные соединения для закрытия
при остановке сервиса. Установка TCP и CONNECT handshake ограничены 10 секундами;
ошибка даёт `502`, таймаут — `504`. CONNECT к `local`, `ygg`, `overspace` и любой
CONNECT на `httpin` получает `405`.

HTTP CONNECT и SOCKS5 CONNECT — разные операции. SOCKS5 используется forwarding
для открытия TCP через Yggstack в overlay-маршрутах; после handshake по потоку
идут обычные HTTP-байты либо TLS-байты CONNECT-клиента.
