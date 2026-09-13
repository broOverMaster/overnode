## Development

- Format Go code with `gofmt`.
- Run `go test ./...` before completing changes.
- Before closing an implementation stage, run `go mod tidy` for every changed Go module and `gofmt` for all Go source code.
- Keep packages focused and avoid unnecessary dependencies.
- Add code, configuration and operational behavior to `common` only when it has a demonstrated need in multiple services. Service-specific behavior and settings belong to that service.
- If a request would move service-specific behavior or settings into `common`, explicitly warn about the boundary violation before making the change, even when the request asks for it.
