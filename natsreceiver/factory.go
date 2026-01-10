package natsreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("nats")
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelDevelopment),
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Endpoint:           "http://localhost:8222",
		CollectionInterval: 60 * time.Second,
		GetVarz:            MetricFilter{Enabled: true},
		StartupLog:         true,
		ConfigLog:          true,
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	metricsConsumer consumer.Metrics,
) (receiver.Metrics, error) {
	return newNatsReceiver(params, cfg.(*Config), metricsConsumer, nil), nil
}

func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	cfg component.Config,
	logsConsumer consumer.Logs,
) (receiver.Logs, error) {
	return newNatsReceiver(params, cfg.(*Config), nil, logsConsumer), nil
}
