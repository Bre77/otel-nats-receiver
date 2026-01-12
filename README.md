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

```yaml
receivers:
  nats:
    # NATS server monitoring endpoint (required)
    endpoint: http://localhost:8222

    # How often to collect metrics (default: 60s)
    collection_interval: 30s

    # Core server metrics
    varz: true              # General server stats (default: true)
    connz: true             # Connection info
    connz_detailed: false   # Detailed connection info (higher overhead)
    routez: false           # Route info for clustered servers
    subz: false             # Subscription info
    leafz: false            # Leaf node info
    gatewayz: false         # Gateway info for super-clusters

    # Health check metrics
    healthz: false                  # Basic health status
    healthz_js_enabled_only: false  # JetStream enabled health
    healthz_js_server_only: false   # JetStream server health

    # Account metrics
    accstatz: false         # Account stats
    accountz: false         # Account info

    # JetStream metrics (string, not boolean)
    # Options: "all", "streams", "consumers", or specific stream name
    # Leave empty or omit to disable
    jsz: "all"

    # Server identification (useful when monitoring multiple servers)
    use_internal_server_id: false   # Use server_id from /varz as identifier
    use_internal_server_name: true  # Use server_name from /varz as identifier

    # Log emission (requires logs pipeline)
    startup_log: true       # Emit log on initial connection
    config_log: true        # Emit log when server config reloads
```

### Metric Filtering

Each endpoint (except `jsz`) can be configured as:

- `true` - Collect all metrics from this endpoint
- `false` - Disable this endpoint
- `["metric1", "metric2"]` - Collect only specific metrics (suffix only)

Example with filtering:

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

    # Disable route metrics
    routez: false
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
