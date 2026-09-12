# Отложенные задачи

Журнал открытых проблем и отложенных решений. Как только задача закрыта и исправление
попало в коммит, её описание удаляется из файла: здесь остаётся только то, что ещё
требует работы. Идентификаторы не переиспользуются, следующий свободный ID — `T-3`.

Закрытие подтверждается коммитом; в его сообщении указывайте ID задачи (например,
`T-1`), чтобы связь сохранялась в истории.

Формат записи:

- **ID** — сквозной номер вида `T-6`;
- **Дата** — когда зафиксировали;
- **Критичность** — `высокая`, `средняя`, `низкая`;
- **Статус** — `открыта` или `в работе`;
- **Где** — файлы и функции;
- **Решение** — что делаем, если уже понятно.

## Сводка

| ID | Дата | Тема | Критичность | Статус |
| --- | --- | --- | --- | --- |
| T-2 | 2026-09-13 | Resolve non-key overspace names | низкая | открыта |

## T-2. Resolve non-key overspace names

- **Date:** 2026-09-13
- **Severity:** low
- **Status:** open
- **Location:** planned `overgate/internal/proxyf` overspace route.

### Current scope

The initial overspace route accepts only `<hex_public_key>.overspace`, derives
the Yggdrasil IPv6 address, and forwards HTTP through the configured Yggstack
SOCKS5 proxy. Other overspace names return HTTP 400. `local.overspace` is handled
separately by the local route.

### Deferred work

Resolve an overspace target that does not match `<hex_public_key>.overspace`
to a public key before deriving the destination IPv6 address. Specify the
resolver source/protocol, trust rules, caching, and lookup failure behavior in
a separate design. Keep overspace dispatch distinct from ygg so this can be
added without changing ygg routing.

### Acceptance

A non-key overspace name resolves to the expected public key and reaches its
HTTP destination through SOCKS5. Literal-key names and local.overspace retain
their behavior. Lookup failures do not fall back to public DNS or internet.
Remove this entry only after the resolver lands in a commit referencing T-2.

## Шаблон записи

~~~~md
## T-<N>. <Тема>

- **Дата:** YYYY-MM-DD
- **Критичность:** низкая
- **Статус:** открыта
- **Где:** [`path/to/file.go`](../path/to/file.go)

### Что происходит

<факты и воспроизведение>

### Насколько критично

<влияние на работу и эксплуатацию>

### Варианты решения

1. <вариант>
~~~~
