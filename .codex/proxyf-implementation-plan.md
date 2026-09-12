# Overgate HTTP proxy implementation plan

Status: stage 1 implemented and prepared for commit at the user's request.
Stage 2 has not started; wait for explicit authorization to proceed.
Target branch: `f/proxyf`.

Execution rule: stop after each stage and wait for explicit user confirmation
of its completion before starting the next stage.

## Agreed scope

Implement the forward HTTP proxy in `overgate/internal/proxyf`.
Use `net/http.Server` to accept connections and parse HTTP/1.x requests; its
handler implements proxy behavior. Origin-form requests are not supported.
There is no website or application API served by this listener.

Routing uses the target in the request line, never the `Host` header as a
fallback. Parse the hostname separately from the port before classifying it.

| Target hostname | Route | Behavior |
| --- | --- | --- |
| `local.overspace` | `local` | Forward HTTP to `proxyf.local_site`, with `Host: local.overspace`. |
| Any other hostname ending in `.overspace` | `overspace` | Initially resolve a hex public key to a Yggdrasil IPv6 address and connect through SOCKS5. |
| Hostname ending in `.ygg` | `ygg` | Resolve a hex public key to a Yggdrasil IPv6 address and connect through SOCKS5. |
| Everything else, including IP literals | `internet` | Connect directly, or through the configured upstream HTTP proxy. |

- Ordinary proxy requests use absolute-form, for example
  `GET http://example.com/path HTTP/1.1`.
- Origin-form requests, such as `GET /path HTTP/1.1`, return 400.
- CONNECT uses authority-form and is supported only for `internet`.
  CONNECT for `local`, `ygg`, or `overspace` returns 405 before key resolution.
- Preserve an explicitly supplied destination port. Default ordinary HTTP
  destinations to port 80.
- A malformed public key for `ygg` or `overspace` returns 400.
- `overspace` remains a distinct route even while sharing key resolution and
  transport with `ygg`. Human-readable name resolution is deferred as T-2.
- Use the agreed configuration prefix `proxyf`: `proxyf.http_proxy`,
  `proxyf.yggstack_proxy`, and `proxyf.local_site`, matching the package name.
- Follow the existing oversite lifecycle: entrypoint-owned configuration,
  logging and signal context; app.Run constructs the component with New and
  calls its blocking Run(ctx); context cancellation closes the server.

## Proposed implementation decisions

These details fill gaps in the agreed contract and must be reviewed when their
implementation stage starts. They are not previously agreed requirements.

- Use the user-agreed `proxyf.listen_on` default `:2080` (all interfaces).
  Accept an empty host or a numeric IP; publish development ports on host loopback.
- Accept only `http://` absolute-form URLs initially. HTTPS clients use CONNECT
  on the internet route. Return 400 for unsupported schemes, URL credentials,
  fragments, invalid authorities, invalid ports, and unsupported request forms.
- Require an explicit valid port in CONNECT authority-form, as required by that
  request form. The ordinary HTTP default of 80 does not infer a CONNECT port.
- Compare DNS hostnames case-insensitively and normalize a single trailing dot.
  Match whole TLD labels; `x.ygg.example` and `notlocal.overspace` must not be
  mistaken for `ygg` and `local`, respectively.
- Represent `proxyf.http_proxy` as an `http://host:port` URL,
  `proxyf.yggstack_proxy` as a `socks5://host:port` URL, and
  `proxyf.local_site` as an `http://host[:port]` URL without path/query/userinfo.
  Reject unsupported forms at startup. Upstream authentication is outside this
  initial scope unless explicitly added during review.
- Permit empty local/SOCKS5 settings so internet-only use remains possible;
  requests needing an unconfigured route return 503. Invalid nonempty settings
  fail startup. Unreachable upstreams return 502; upstream timeouts return 504
  when an HTTP response can still be sent. Never fall back to another route.
- For ygg/overspace preserve the original logical target authority in the
  outgoing Host header while dialing the computed IPv6 and destination port.
  For local always use exactly `Host: local.overspace`.
- Do not obtain proxy settings from HTTP_PROXY/HTTPS_PROXY environment variables
  implicitly; only the application's configured upstream controls the route.

## Component boundaries

- `overgate/cmd/overgate`: configuration loading, logger setup, signals, exit code.
- `overgate/internal/app`: aggregate configuration and component lifecycle.
- `overgate/internal/proxyf`: configuration/schema, target parsing, routing,
  forwarding, CONNECT handling, Yggdrasil address derivation, shutdown.
- Reuse `common/pkg/config`, `common/pkg/config/schema`, and
  `common/pkg/logging`. Keep proxy-specific behavior and settings in overgate.
- Start with focused files inside proxyf, not a hierarchy of one-function
  packages. Share forwarding primitives without merging route identities.

## Stage 1 — Establish the service lifecycle and configuration

- [x] Replace the greeting entrypoint with a testable run function following
  the existing oversite lifecycle pattern: config.Load, logging.New with deferred
  cleanup, signal.NotifyContext for SIGINT/SIGTERM, app.Run, and exit status
  0 for help/normal stop or 1 for configuration/startup/runtime errors.
- [x] Keep app.Run as thin wiring: proxyf.New(configuration, logger), then
  server.Run(ctx). New validates dependencies/configuration; Run opens the TCP
  listener and owns runtime resources. Use a listener-taking serve helper for
  tests, following oversite/internal/httpd.
- [x] Add app configuration and the proxyf configuration schema.
- [x] Validate the listener and upstream addresses before accepting requests.
- [x] Configure listener header/idle timeouts, header limits and request context
  cancellation. Transport creation/cleanup belongs to stage 3, where forwarding
  is introduced; no unused transports are instantiated in stage 1.
- [x] Add structured service/error logging using existing common logging;
  do not log credentials, request bodies, or URL query strings. Route-specific
  logging follows route dispatch in stage 2.
- [x] Match oversite's stop behavior: on context cancellation call
  http.Server.Close, await Serve completion, treat http.ErrServerClosed as
  normal, and log start/stop. Do not introduce a graceful-drain phase.
  Ensure cleanup also runs on Serve failure and partially completed startup.
  Outbound operation/transport cleanup is added in stage 3; tracked CONNECT
  tunnel cleanup is added in stage 4 because Close does not manage hijacks.
- [x] Test help, invalid configuration, configuration precedence, startup errors,
  and cancellation. A running process alone is not functional acceptance.

Stage 1 acceptance: `make check` and `go test -race ./overgate/...` must pass.
Run `.local/bin/overgate --proxyf.listen_on=127.0.0.1:2080`; a request to the
listener returns the temporary 501 response. SIGINT/SIGTERM stops the listener
and exits successfully. No route is implemented or accepted at this stage.

## Stage 2 — Parse and classify proxy requests

- [ ] Parse request-target and validate its form before dispatch.
- [ ] Separate normalized routing hostname, original authority, destination port,
  path, and query so dialing changes do not corrupt request semantics.
- [ ] Implement exact local matching and label-aware TLD classification.
- [ ] Reject origin-form with 400 and special-route CONNECT with 405.
  Supply a suitable Allow header for 405, listing supported ordinary methods.
- [ ] Test mixed case, trailing dots, explicit/default ports, IPv4/IPv6 literals,
  lookalike suffixes, malformed URLs, conflicting Host headers, and unsupported
  forms. Confirm there is no fallback to Host or DNS for malformed key routes.

## Stage 3 — Make local and direct internet HTTP work

- [ ] Instantiate long-lived transports once with bounded dial/header timeouts;
  cancel outbound operations and close idle connections on all exit paths.
- [ ] Use http.Transport.RoundTrip for forwarding; clear outbound RequestURI.
  Do not use a redirect-following client.
- [ ] Preserve method, escaped path, query, body, status, and end-to-end headers.
  Strip hop-by-hop headers in both directions, including headers nominated by
  Connection; keep proxy credentials from leaking to origin servers.
- [ ] Stream request/response bodies with cancellation and close response bodies.
  Define trailer handling explicitly; do not silently break HTTP framing.
- [ ] For local, connect to local_site and set Host to local.overspace. Do not
  resolve local.overspace through DNS or send it through the internet proxy.
- [ ] For internet without an upstream, dial the target directly, with no
  implicit environment proxy.
- [ ] Test GET and POST against controlled origins, redirects without following
  them, body integrity, Host/path/query, backend failures, and route isolation.
- [ ] Run acceptance gate A below. Both routes must work before moving on.

## Stage 4 — Add internet proxy chaining and CONNECT

- [ ] Configure the internet HTTP transport to use proxyf.http_proxy when set.
  Verify the upstream receives absolute-form HTTP requests.
- [ ] For direct CONNECT, establish the destination TCP connection before
  sending 200 Connection Established and starting bidirectional copying.
- [ ] With an upstream proxy, send CONNECT to that proxy and require a successful
  response before acknowledging the client. Handle failures without entering
  tunnel mode; do not dial the origin directly as a fallback.
- [ ] Preserve already buffered bytes on both client and upstream sides when
  switching to tunnel mode. Handle EOF, half-close where supported, cancellation,
  bounded shutdown, and connection cleanup without goroutine leaks.
- [ ] Keep TLS end-to-end; no TLS interception or certificate generation in the
  proxy. Test HTTPS with a trusted local test CA.
- [ ] Test direct/chained HTTP and HTTPS, upstream refusal/non-2xx responses,
  early buffered tunnel data, large bidirectional transfers, and shutdown with
  active tunnels. Reject all special-route CONNECT cases with 405.
- [ ] Run acceptance gate B below.

## Stage 5 — Implement ygg and initial overspace routing

- [ ] Verify the supported Yggdrasil/Yggstack version and authoritative public
  key-to-IPv6 derivation API. Prefer a maintained upstream implementation;
  inspect dependency scope before choosing a package. Do not invent a hash rule.
- [ ] Validate the entire prefix as one hex-encoded public key of the exact
  length required by that version (expected Ed25519: 32 bytes / 64 hex digits).
- [ ] Add independently obtained known key/address vectors, invalid lengths,
  non-hex input, and extra-label tests; avoid testing an algorithm against itself.
- [ ] Dial the derived IPv6 and original/default port through SOCKS5 using an
  IPv6 destination address. Never ask DNS or SOCKS5 to resolve the .ygg or
  .overspace name, and never use the internet upstream for these routes.
- [ ] Keep separate route dispatch for overspace while reusing derivation/dialing.
- [ ] Use a controlled SOCKS5 test server to verify the exact destination bytes,
  port, forwarded HTTP semantics, cancellation, and failed handshakes.
- [ ] Keep T-2 open; non-key overspace names return 400 in this first version.

## Stage 6 — Build a reproducible Docker acceptance environment

- [ ] Add a dedicated test Compose file and fixture scripts/configuration, with
  exact start/check/stop commands documented alongside them. Keep test services
  opt-in and separate from normal service startup and private .local settings.
- [ ] Pin Yggstack to a verified version/image digest or build a pinned upstream
  revision. Verify its actual SOCKS5 flags, configuration, readiness mechanism,
  and network requirements before writing the Compose service.
- [ ] Run a source Yggstack exposing SOCKS5 and a reachable destination node
  serving an HTTP fixture over its Yggdrasil IPv6. A SOCKS5 listener alone is
  insufficient to verify successful overlay routing.
- [ ] Prefer two deterministic test nodes peered directly over a Docker network,
  avoiding reliance on public peers. Determine whether the destination requires
  TUN/NET_ADMIN or an upstream-supported userspace listener; document and scope
  any required privileges to the test fixture.
- [ ] Use test-only identities, record the destination public key/IPv6, and
  provide an HTTP endpoint on port 80 plus another port for explicit-port tests.
- [ ] Include a controlled internet HTTP/HTTPS origin and upstream HTTP proxy
  with connection/request evidence to prove direct versus chained behavior.
- [ ] Include oversite with a known static response. Its listener is loopback-only:
  a separate container cannot reach it via backend DNS alone. Use a shared
  network namespace with overgate for the container fixture, or run both on host
  loopback for the local acceptance stage. Do not relax oversite's contract.
- [ ] Publish only client-facing test ports on host loopback. Keep fixture keys
  distinct from real identities; preserve useful failure logs before teardown.
- [ ] Run gate C. If Docker/Yggstack is unavailable, report overlay acceptance
  as not run; unit tests and SOCKS5 mocks do not substitute for this gate.

## Stage 7 — Documentation and final verification

- [ ] Update Russian overgate/root documentation with supported forms, route
  rules, configuration, startup examples, CONNECT limits, and error behavior.
- [ ] Document Docker acceptance prerequisites, pinned versions, commands,
  expected results, and teardown. Keep logs/help/errors and .codex context English.
- [ ] Run gofmt and `make check` across all modules. The workspace equivalent
  of per-module `go test ./...` is `make test`; use module-local tests as needed.
- [ ] Run `go test -race ./overgate/...` for connection lifecycle/concurrency and
  the explicit Docker integration suite. Re-run affected checks after fixes.
- [ ] Record acceptance evidence and remaining limitations in this plan.
  Do not close T-2 until its resolver is implemented and committed.

## Acceptance procedure and gates

Use controlled response markers and request/connection records as evidence.
Every client invocation must explicitly use the proxy and disable NO_PROXY
bypass, e.g. `curl --noproxy '' --proxy http://127.0.0.1:2080 ...`.
Replace example ports with the fixture's documented published ports.

### Gate A — Required working local and direct internet routes

1. Start oversite with a known marker file and configure local_site to its
   reachable loopback endpoint. Start overgate with no internet upstream.
2. Request `http://local.overspace/marker.txt` through overgate; require 200 and
   the exact marker. Use a recording HTTP backend as a separate check to prove
   Host is exactly local.overspace and path/query/method/body are preserved.
3. Request a controlled ordinary hostname or IP HTTP origin through overgate;
   require the expected response and evidence of a direct origin connection.
   Use an explicitly addressed fixture port and also test default destination 80.
4. Send `curl http://127.0.0.1:2080/` without --proxy; require 400.
5. Stop each backend in turn; require a bounded 502/504 response, no process
   crash, and successful requests again after backend recovery.

### Gate B — Complete internet behavior and routing boundaries

| Check | Required result |
| --- | --- |
| Internet HTTP, upstream configured | Expected response; upstream records the absolute-form request. |
| Internet HTTPS, no upstream | CONNECT succeeds; TLS and response verified with test CA. |
| Internet HTTPS, upstream configured | Upstream records CONNECT to the target; end-to-end TLS succeeds. |
| Upstream stopped or rejecting CONNECT | Bounded failure; no direct fallback. |
| CONNECT local.overspace:443 | 405; no local backend connection. |
| CONNECT key.ygg:443 / key.overspace:443 | 405; no SOCKS5 connection, irrespective of key validity. |
| Malformed request-target or origin-form | 400. |
| Host header differs from request-line target | Route comes from request line; Host cannot change it. |
| SIGTERM during active HTTP/CONNECT traffic | Shutdown finishes within its documented bound; no stranded tunnels. |

### Gate C — Real ygg and overspace through Docker Yggstack/SOCKS5

1. Start the source SOCKS5/Yggstack and destination HTTP node; wait for overlay
   connectivity and demonstrate direct SOCKS5 access to the destination IPv6.
2. Configure proxyf.yggstack_proxy to the source SOCKS5 listener. Request
   `http://<destination_public_key>.ygg/marker` through overgate; require the
   destination marker and evidence that the derived IPv6 was reached via SOCKS5.
3. Repeat with `<destination_public_key>.overspace`; require the same backend
   response with a distinct overspace route in diagnostic evidence.
4. Repeat both suffixes with default port 80 and an explicit alternate port.
   Verify the original logical Host reaches the destination.
5. Send wrong-length/non-hex/multi-label key names and a human-readable
   overspace name; require 400 with no outbound connection attempt.
6. Stop SOCKS5, then separately break overlay reachability; require bounded
   failures and no fallback to internet/DNS. Local and internet must still work.
7. Re-run a local and internet smoke check with all route settings populated
   to verify transport isolation.

Gate A is the minimum usable milestone explicitly required by the user. Gate B
completes internet acceptance. The full implementation is accepted only after
all three gates pass; an unavailable overlay environment is a recorded pending
acceptance step, not a successful result.

## Evidence to record after implementation

Stage 1 verification: `make check` passed after changing the default listener
to `:2080`. `go test -race ./overgate/...` passed for the lifecycle implementation
before that default-only adjustment. Tests cover the temporary 501 response,
listener closure on cancellation, listen/serve failures, configuration validation,
help, and environment/CLI precedence. Gates A/B/C remain pending because routing
has not been implemented. No Docker acceptance services have been started.

The user-created `.local/etc/overgate.config.yaml` is currently empty and is not
part of the implemented configuration examples; preserve it as local user work.

- Commit/worktree revision, Go and Docker versions, fixture image revisions.
- Commands used, sanitized configuration, test target public key and IPv6.
- Gate A/B/C pass/fail/not-run results, response markers, relevant route/upstream
  records, and reasons for any pending checks.
- `make check`, race-test and integration-test results; cleanup outcome.
