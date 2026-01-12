# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a custom OpenTelemetry Collector receiver for NATS. It scrapes metrics from NATS server HTTP monitoring endpoints, converts them from Prometheus format to OpenTelemetry metrics, and feeds them into the OTel collector pipeline. It also emits OTel logs on startup and config reload events.

## Build Commands

```bash
# Build the collector (generates code in build/ and compiles binary)
./ocb --config builder-config.yaml

# Build with verbose output
./ocb --config builder-config.yaml --verbose

# Generate code only, skip compilation
./ocb --config builder-config.yaml --skip-compilation
```

## Test Commands

```bash
# Run all tests
cd natsreceiver && go test ./...

# Run specific test
cd natsreceiver && go test -run TestGetServerID

# Run with verbose output
cd natsreceiver && go test -v ./...

# Run with coverage
cd natsreceiver && go test -cover ./...
```

## Running the Collector

```bash
# Run with example config (see example/config.yaml for full options)
./build/otelcol-nats --config example/config.yaml
```

## Architecture

**Core package: `natsreceiver/`**

- `factory.go` - Creates the OTel receiver factory, registers component type `nats`
- `config.go` - Configuration structs including `MetricFilter` (flexible bool/list type for filtering)
- `receiver.go` - Main implementation:
  - `natsReceiver` implements the receiver interface, handles both metrics and logs pipelines
  - `natsScraper.initCollectors()` sets up Prometheus collectors based on enabled config options
  - `natsScraper.Scrape()` collects metrics and transforms them to OTel format
  - `transformMetricName()` converts `gnatsd_*` prefixes to `nats.*` OTel naming
  - `shouldCollectMetric()` applies metric filters per endpoint
  - `emitLog()` creates OTel log records for startup and config reload events

**Data flow:**
```
NATS HTTP endpoints → Prometheus collectors → Scrape() → OTel metrics → Exporter pipeline
                   → fetchVarz() → emitLog() → OTel logs → Exporter pipeline
```

**Configuration types:**
- `MetricFilter` - Most endpoints use this flexible type that accepts either:
  - `true`/`false` - enable/disable all metrics from endpoint
  - `["metric1", "metric2"]` - collect only specific metrics (suffix only, e.g., `["cpu", "mem"]` for varz)
- `GetJsz` (string) - JetStream uses a different pattern: `"all"`, `"streams"`, `"consumers"`, or specific stream name

**Supported endpoints:** varz, connz, connz_detailed, routez, subz, leafz, gatewayz, healthz, healthz_js_enabled_only, healthz_js_server_only, accstatz, accountz, jsz

## Key Dependencies

- OpenTelemetry Collector SDK v0.143.0
- `github.com/nats-io/prometheus-nats-exporter` - NATS Prometheus collectors
- `github.com/prometheus/client_golang` - Prometheus metric types
