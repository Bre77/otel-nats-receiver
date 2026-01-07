// Package natsreceiver implements an OpenTelemetry Collector receiver that
// collects metrics from NATS server monitoring endpoints.
//
// The receiver scrapes metrics from the NATS server's HTTP monitoring interface
// (typically available at port 8222) and converts them to OpenTelemetry format.
//
// # Configuration
//
// The receiver supports the following configuration options:
//
//   - endpoint: URL of the NATS server monitoring endpoint (default: http://localhost:8222)
//   - collection_interval: How often to collect metrics (default: 60s)
//   - varz: Collect general server stats (default: true)
//   - connz: Collect connection info
//   - routez: Collect route info
//   - jsz: Collect JetStream metrics (set to filter like "all", "streams", etc.)
//
// # Usage
//
// To use this receiver, build a custom OpenTelemetry Collector using the
// OpenTelemetry Collector Builder (ocb) with the provided builder-config.yaml,
// or import the receiver directly:
//
//	import "github.com/Bre77/otel-nats-receiver/natsreceiver"
//
//	factory := natsreceiver.NewFactory()
package natsreceiver
