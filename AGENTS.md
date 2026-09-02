# AGENTS.md

Guidance for agents working in this repository.

## Project Overview

A custom OpenTelemetry Collector receiver (component type `nats`) that scrapes
NATS server HTTP monitoring endpoints, converts the Prometheus-format metrics to
OTel metrics, and emits OTel logs on server startup and config reload. User-facing
configuration, endpoint list, metric naming and emitted attributes are documented
in `README.md` - keep it in sync with `natsreceiver/config.go` rather than
restating it here.

## Commands

```bash
./ocb --config builder-config.yaml   # generate build/ and compile the collector
./build/otelcol-nats --config example/config.yaml
cd natsreceiver && go test ./...     # also: go vet ./..., go build ./...
```

`ocb` and `build/` are gitignored - download the
[collector builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder)
into the repo root before building.

## Architecture

All code lives in `natsreceiver/`: `factory.go` (receiver factory and defaults),
`config.go` (config structs plus the `MetricFilter` bool/list/object unmarshalling),
`receiver.go` (`natsReceiver` for lifecycle and the logs pipeline, `natsScraper`
for metrics).

```
NATS HTTP endpoints → Prometheus collectors → Scrape() → OTel metrics → pipeline
                   → fetchVarz() → emitLog()           → OTel logs    → pipeline
```

Most endpoint options are `MetricFilter` (accepts `true`/`false`/list of metric
name suffixes). `jsz` is the exception: a plain string (`"all"`, `"streams"`,
`"consumers"`, or a stream name), so it is not in `metricFilterFields` and does
not go through the same conversion path.

## Resource Metadata Conventions

Both metrics (`natsScraper.Scrape()`) and logs (`natsReceiver.emitLog()`) must emit
the same resource identity: `service.name`, `service.instance.id` (server ID),
`host.name` (server name), `service.version` - all sourced from `/varz` fetched
once at `Start()` and cached on `natsReceiver` (`serverID`/`serverName`/`serverVersion`).
Never read these per-scrape from `/varz`: a scrape must not block on a network call
for identity that cannot change during the receiver's lifetime.

`deployment.environment.name` is set only when the optional `environment` config
field is non-empty - never hardcode an environment value in receiver code. A generic
receiver cannot infer environment from the target system; that belongs to the
deployment config.

## CI and releases

`.github/workflows/ci.yml` runs vet/build/test in `natsreceiver/` on push/PR to
`main`. `.github/workflows/release.yml` is manually dispatched with a `version`
input: it re-runs the same gate, then a second job gated on the `production` GitHub
Environment (required reviewer approval) tags that commit and cuts the release.
Tagging must only ever happen inside that gated job - there is no push-to-tag
trigger, so a release cannot be cut without passing CI on the exact commit and
getting approval. Both workflows pin the Go toolchain to `natsreceiver/go.mod`.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
