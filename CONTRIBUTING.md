# Contributing

Thank you for your interest in Ploos AI Gateway.

This project is at milestone M0. Please discuss major design changes in an issue before implementation.

## Local checks

```sh
gofmt -w cmd/gateway/main.go internal/api/*.go
go vet ./...
go test -race ./...
go build ./cmd/gateway
```

Keep the core independent of any specific upstream service, never commit secrets, and include tests for API behavior. Contributions are submitted under the repository's MIT license.
