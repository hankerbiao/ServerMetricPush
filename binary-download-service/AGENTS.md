# Repository Guidelines

## Project Structure & Module Organization
This repository is a Go 1.21 service that starts `node_exporter`, scrapes local metrics, and pushes them to Prometheus Pushgateway. Keep application code under `src/`:

- `src/main.go`: CLI entrypoint and process lifecycle
- `src/config/`: config parsing, validation, and defaults
- `src/pusher/`: Pushgateway client logic and tests

Operational assets live at the repository root: `config.yml` for local and install-time configuration, `systemd/node-push-exporter.service` for Linux service setup, and `README.md` for usage and deployment examples.

## Build, Test, and Development Commands
Use the standard Go toolchain from the repository root:

- `go build -o node-push-exporter ./src`: build the service binary
- `go run ./src -config ./config.yml`: run locally with an explicit config file
- `go test ./...`: run the full unit test suite
- `go test ./src/pusher -run TestPusher`: run Pushgateway-focused tests only
- `go fmt ./...`: format all Go files before review

For manual verification, use `node-push-exporter --version` and `curl http://localhost:9100/metrics` after startup.

## Coding Style & Naming Conventions
Keep code `gofmt`-clean and rely on standard Go formatting, including tab indentation. Use lowercase package names such as `config` and `pusher`. Exported identifiers should use `CamelCase`; internal helpers should use `camelCase`. Prefer small, package-local functions over shared utility files with mixed responsibilities.

## Testing Guidelines
Write table-driven tests when behavior branches by input or configuration. Place tests beside the code they cover in `*_test.go` files. Follow the pattern `Test<Type>_<Scenario>`, for example `TestPusher_Push_Success`. New changes should cover config defaults, HTTP error handling, and Pushgateway path construction when applicable.

## Commit & Pull Request Guidelines
Local Git history is not available in this checkout, so use short imperative commit subjects such as `fix pusher timeout handling` or `add metrics url config`. Pull requests should include a clear summary, linked issue when relevant, test evidence such as `go test ./...`, and notes for any config or `systemd` changes.

## Configuration & Deployment Tips
Do not hardcode production Pushgateway URLs, credentials, or instance labels. Keep deployable defaults in `config.yml`, and update `systemd/node-push-exporter.service` whenever startup flags, paths, or runtime assumptions change.
