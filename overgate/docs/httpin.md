# Приём HTTP из overlay

`internal/httpin` — HTTP-сервер для запросов, доставленных Yggstack или другим
доверенным входом. Это не forward proxy: он принимает только origin-form.

```http
GET /path?q=1 HTTP/1.1
Host: <key>.overspace
```

```text
Yggstack -remote-tcp
  =(HTTP / TCP)=> httpin.listen_on
  → httpin.Server
  → httputil.ReverseProxy
  =(HTTP / TCP)=> httpin.local_site
  → oversite/httpd
```

`httpin` всегда пересылает запрос в один backend из `httpin.local_site`. Host,
метод, path/query, тело, сквозные заголовки и трейлеры сохраняются; Host не
запускает повторную маршрутизацию через `proxyf`.

В приёмочном Compose-стенде `httpin` слушает `127.0.0.1:8080` в сетевом
пространстве контейнера второго Yggstack. Его `-remote-tcp 80:127.0.0.1:8080`
доставляет overlay TCP-поток непосредственно на этот listener. Backend задан как
`overlay-site:8000`.

Origin-form с ошибкой получает `400`, CONNECT — `405`, ошибка backend — `502`,
таймаут соединения или заголовков — `504`. Пустой `httpin.listen_on` отключает
listener, но полезный handler компонента сохраняется для встраивания.
