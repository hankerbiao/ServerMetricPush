# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go 1.21 service that launches `node_exporter`, scrapes local metrics, and pushes them to Prometheus Pushgateway. Source code lives under `src/`:

- `src/main.go`: CLI entrypoint and process lifecycle
- `src/config/`: config parsing and defaults
- `src/pusher/`: Pushgateway client and tests

Operational assets live outside `src/`: `config.yml` for local runs or as the source template for system installs, `systemd/node-push-exporter.service` for Linux service setup, and `README.md` for install and runtime examples.

## Build, Test, and Development Commands
- `go build -o node-push-exporter ./src`: build the binary from the main package
- `go run ./src -config ./config.yml`: run locally with an explicit config file
- `go test ./...`: run all unit tests across packages
- `go test ./src/pusher -run TestPusher`: run only Pushgateway client tests
- `go fmt ./...`: apply standard Go formatting before review

Use `node-push-exporter --version` to verify the built binary and `curl http://localhost:9100/metrics` to confirm the exporter endpoint is reachable.

## Coding Style & Naming Conventions
Follow standard Go formatting and keep code `gofmt`-clean. Use tabs as emitted by `gofmt`, package names in lowercase (`config`, `pusher`), exported identifiers in `CamelCase`, and unexported helpers in `camelCase`. Keep new files near the package they belong to, and prefer small package-focused functions over cross-package utility dumping.

## Testing Guidelines
Write table-driven tests where behavior branches. Keep tests in `*_test.go` beside the code they cover, and name them `Test<Type>_<Scenario>` as in `TestPusher_Push_Success`. Cover config parsing defaults, HTTP error handling, and Pushgateway path construction when adding features.

## Commit & Pull Request Guidelines
This checkout does not include `.git` history, so no local commit pattern could be verified. Use short imperative commit subjects such as `fix pusher timeout handling` or `add metrics url config`. PRs should include a clear summary, linked issue if applicable, test evidence (`go test ./...`), and config or service-file notes when behavior changes affect deployment.

## Configuration & Deployment Tips
Do not hardcode production Pushgateway URLs or instance labels. Keep deployable defaults in `config.yml`, and update `systemd/node-push-exporter.service` whenever startup flags or binary paths change.
