# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`node-push-exporter` is a Go 1.21 service that launches `node_exporter` as a child process, scrapes system metrics from its `/metrics` endpoint, and pushes them to Prometheus Pushgateway at configurable intervals.

## Commands

```bash
# Build the binary
go build -o node-push-exporter ./src

# Run locally with explicit config
go run ./src -config ./config.yml

# Run all unit tests
go test ./...

# Run only Pushgateway client tests
go test ./src/pusher -run TestPusher

# Format code
go fmt ./...

# Verify binary version
node-push-exporter --version
```

## Architecture

The codebase follows a simple, layered architecture:

- **src/main.go**: CLI entrypoint, process lifecycle, spawns/manages node_exporter subprocess, handles signal-based graceful shutdown
- **src/config/config.go**: Configuration loading from key=value format files with required field validation
- **src/pusher/pusher.go**: Pushgateway client using functional options pattern, constructs URL paths like `/metrics/job/<job>/instance/<instance>`

Configuration uses `key=value` format (not YAML/JSON) for easier shell script generation. The config loader enforces required fields at load time—no silent defaults that could mask deployment misconfiguration.

## Key Behaviors

- node_exporter is spawned as a managed child process; it stops when the main program exits
- Metrics are scraped via HTTP immediately after node_exporter readiness is confirmed (not waiting for first interval)
- HTTP timeouts are set on both the scrape (from node_exporter) and push (to Pushgateway) operations
- The pusher uses Prometheus text format version 0.0.4 as required by Pushgateway

## Code Organization

- Tests live in `*_test.go` files alongside the code they cover
- Table-driven tests used for multi-scenario coverage (see pusher_test.go)
- Config validation happens in `validate()` before returning from `Load()`
