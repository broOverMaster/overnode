# Overgate lifecycle

- `proxyf.New` and `httpin.New` accept `(config, logger, *[]lifecycle.LifeCycle)`
  and return `(http.Handler, error)`. Constructors validate before registration;
  errors leave the registry unchanged. The slice pointer must not be nil.
- Enabled services append themselves to the shared slice. Empty
  `httpin.listen_on` skips registration and logs the reason, but still returns
  a usable handler when `httpin.local_site` is configured. Proxyf listener
  validation and defaults are unchanged.
- `app.Run` initializes all components before calling `lifecycle.Start` and
  `Group.Wait`. The registry contains only non-nil services.
- Start creates a child context and launches registered services. Wait is called
  once; the first completion cancels siblings, all results are collected, and
  errors are joined. Empty groups return immediately. Parent context cancellation
  stops services; child cancellation does not cancel the parent.
- Components attach their names to Run errors and log readiness after binding.
  Main logs application-level failures as `overgate failed`.
- This abstraction stays in `overgate/internal/lifecycle`; common and oversite
  are unchanged. Routing and Docker acceptance configuration are unchanged.
- Verification: `make check`, race tests for overgate, and focused lifecycle/app
  race tests passed during this refactor. Tests cover registration, disabled
  handlers, initialization failures, empty groups, cancellation, sibling shutdown,
  error aggregation, and listener failures. Docker acceptance was not rerun.
- The pre-existing `go.work.sum` modification is user work, not part of this
  lifecycle refactor.
