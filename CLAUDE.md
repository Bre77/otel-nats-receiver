# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a custom OpenTelemetry Collector receiver for NATS. It scrapes metrics from NATS server HTTP monitoring endpoints, converts them from Prometheus format to OpenTelemetry metrics, and feeds them into the OTel collector pipeline.

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
# Run with example config
./build/otelcol-nats --config example/config.yaml
```

## Architecture

**Core package: `natsreceiver/`**

- `factory.go` - Creates the OTel receiver factory, registers component type `nats`
- `config.go` - Configuration structs including `MetricFilter` (flexible bool/list type for filtering)
- `receiver.go` - Main implementation:
  - `natsReceiver` implements the receiver interface
  - `natsScraper.initCollectors()` sets up Prometheus collectors based on enabled config options (varz, connz, jsz, etc.)
  - `natsScraper.Scrape()` collects metrics and transforms them to OTel format
  - `transformMetricName()` converts `gnatsd_*` prefixes to `nats.*` OTel naming
  - `shouldCollectMetric()` applies metric filters per endpoint

**Data flow:**
```
NATS HTTP endpoints → Prometheus collectors → Scrape() → OTel metrics → Exporter pipeline
```

**Configuration endpoints:** The receiver supports multiple NATS monitoring endpoints (varz, connz, routez, gatewayz, leafz, subz, jsz, accountz, accstatz, healthz) each configurable as boolean or metric filter list.

## Key Dependencies

- OpenTelemetry Collector SDK v0.143.0
- `github.com/nats-io/prometheus-nats-exporter` - NATS Prometheus collectors
- `github.com/prometheus/client_golang` - Prometheus metric types
