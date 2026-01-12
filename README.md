# NATS Receiver for OpenTelemetry Collector

A custom OpenTelemetry Collector receiver that scrapes metrics from NATS server HTTP monitoring endpoints and converts them to OpenTelemetry format.

## Features

- Collects metrics from all NATS monitoring endpoints (varz, connz, routez, jsz, etc.)
- Converts Prometheus-format metrics to OpenTelemetry metrics
- Flexible per-endpoint filtering to collect only the metrics you need
- Emits OTel log records on server startup and configuration reload
- Supports server identification via server_id or server_name

## Installation

Add this receiver to your OpenTelemetry Collector build using the [OpenTelemetry Collector Builder (ocb)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).

### builder-config.yaml

```yaml
dist:
  name: otelcol-custom
  description: Custom OpenTelemetry Collector with NATS receiver
  output_path: ./build
  otelcol_version: 0.143.0

receivers:
  - gomod: github.com/Bre77/otel-nats-receiver/natsreceiver v0.1.0

# Add your exporters, processors, and extensions as needed
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.143.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.143.0
```

Build the collector:

```bash
ocb --config builder-config.yaml
```

## Configuration

### Endpoint Options

Each endpoint option (except `jsz`) accepts one of three value types:

| Value | Behavior |
|-------|----------|
| `true` | Collect all metrics from this endpoint |
| `false` | Disable this endpoint (default for most) |
| `["suffix1", "suffix2"]` | Collect only metrics matching these suffixes |

Example showing all three forms:

```yaml
receivers:
  nats:
    endpoint: http://localhost:8222

    varz: true              # Collect all varz metrics
    connz: false            # Disable connz entirely
    routez:                 # Collect only specific routez metrics
      - in_msgs
      - out_msgs
```

### Basic Configuration

```yaml
receivers:
  nats:
    endpoint: http://localhost:8222
    collection_interval: 30s
    varz: true
    connz: true
    jsz: "all"

service:
  pipelines:
    metrics:
      receivers: [nats]
      exporters: [otlp]
    logs:
      receivers: [nats]
      exporters: [otlp]
```

### Full Configuration Reference

All options with their defaults:

```yaml
receivers:
  nats:
    # NATS server monitoring endpoint (required)
    endpoint: http://localhost:8222

    # How often to collect metrics (default: 60s)
    collection_interval: 60s

    # Core server metrics (all accept true/false/list)
    varz: true                # General server stats (default: true)
    connz: false              # Connection info (default: false)
    connz_detailed: false     # Detailed connection info (default: false)
    routez: false             # Route info for clustered servers (default: false)
    subz: false               # Subscription info (default: false)
    leafz: false              # Leaf node info (default: false)
    gatewayz: false           # Gateway info for super-clusters (default: false)

    # Health check metrics (all accept true/false/list)
    healthz: false                  # Basic health status (default: false)
    healthz_js_enabled_only: false  # JetStream enabled health (default: false)
    healthz_js_server_only: false   # JetStream server health (default: false)

    # Account metrics (all accept true/false/list)
    accstatz: false           # Account stats (default: false)
    accountz: false           # Account info (default: false)

    # JetStream metrics (string only, not boolean or list)
    # Options: "all", "streams", "consumers", or specific stream name
    # Omit or leave empty to disable (default: disabled)
    jsz: ""

    # Server identification (useful when monitoring multiple servers)
    use_internal_server_id: false   # Use server_id from /varz (default: false)
    use_internal_server_name: false # Use server_name from /varz (default: false)

    # Log emission (requires logs pipeline)
    startup_log: true         # Emit log on initial connection (default: true)
    config_log: true          # Emit log when server config reloads (default: true)
```

### Metric Filtering Examples

Filter to collect only specific metrics from an endpoint:

```yaml
receivers:
  nats:
    endpoint: http://localhost:8222

    # Collect only CPU and memory metrics from varz
    varz:
      - cpu
      - mem

    # Collect all connection metrics
    connz: true

    # Collect specific JetStream metrics (use jsz string options instead)
    jsz: "streams"
```

## Metric Naming

Metrics are converted from the NATS Prometheus exporter format to OpenTelemetry naming:

| Original Name | OTel Name |
|--------------|-----------|
| `gnatsd_varz_cpu` | `nats.varz.cpu` |
| `gnatsd_connz_total` | `nats.connz.total` |
| `gnatsd_jsz_streams` | `nats.jsz.streams` |

## Log Records

When `startup_log` or `config_log` are enabled, the receiver emits OpenTelemetry log records with:

**Resource attributes:**
- `service.name`: "nats"
- `service.instance.id`: Server ID
- `service.version`: NATS version
- `host.name`: Server name

**Log attributes:**
- `host`: NATS host
- `port`: NATS port
- `jetstream.enabled`: Whether JetStream is enabled
- `max_connections`: Maximum connections
- `max_payload`: Maximum payload size
- `config_load_time`: Configuration load timestamp
- `cluster.name`: Cluster name (if clustered)

## Requirements

- NATS server with [monitoring enabled](https://docs.nats.io/running-a-nats-service/configuration/monitoring)
- OpenTelemetry Collector v0.143.0+

Enable monitoring in your NATS server config:

```
http_port: 8222
```

Or via command line:

```bash
nats-server -m 8222
```

## License

MIT
