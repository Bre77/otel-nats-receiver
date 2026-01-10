# NATS Receiver Startup Log Feature

## Overview

Add an option to emit OpenTelemetry log records when the receiver connects to a NATS server and when the server's configuration reloads. This assists with service.name discovery and provides visibility into NATS server state.

## Configuration

Two new boolean options, both defaulting to `true`:

```yaml
nats:
  endpoint: http://localhost:8222
  startup_log: true   # Emit log on initial connection
  config_log: true    # Emit log when server config reloads
```

## Varz Fields Captured

Extend the existing varz parsing to capture:

- `server_id` - NATS server unique identifier
- `server_name` - Human-readable server name
- `version` - NATS server version
- `host` - Server host
- `port` - Server port
- `cluster.name` - Cluster name (if clustered)
- `jetstream` - Whether JetStream is enabled
- `max_connections` - Maximum allowed connections
- `max_payload` - Maximum message payload size
- `config_load_time` - When config was last loaded

## Log Record Structure

### Resource Attributes (OTel semantic conventions)

- `service.name` = "nats"
- `service.instance.id` = server_id
- `service.version` = version
- `host.name` = server_name

### Log Attributes

- `host` = host
- `port` = port
- `cluster.name` = cluster name (if present)
- `jetstream.enabled` = jetstream boolean
- `max_connections` = max_connections
- `max_payload` = max_payload
- `config_load_time` = config_load_time (RFC3339 format)

### Log Body

- Startup: "NATS server connected"
- Config reload: "NATS server configuration reloaded"

### Severity

- Level: Info

## Implementation

### Factory Changes

Register the receiver as both a metrics and logs receiver:

```go
func NewFactory() receiver.Factory {
    return receiver.NewFactory(
        typeStr,
        createDefaultConfig,
        receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
        receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
    )
}
```

### Receiver Changes

1. Add `logsConsumer consumer.Logs` field to `natsReceiver`
2. Store `lastConfigLoadTime` for reload detection
3. Create `fetchVarz()` method that returns full varz data
4. Create `emitLog()` method to construct and send log records
5. In `Start()`: fetch varz, emit startup log if enabled
6. In `Scrape()`: if `ConfigLog` enabled, check for config reload

### Config Changes

Add to `Config` struct:

```go
StartupLog bool `mapstructure:"startup_log"`
ConfigLog  bool `mapstructure:"config_log"`
```

Default both to `true` in factory.
