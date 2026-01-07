package natsreceiver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/prometheus-nats-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/zap"
)

type natsReceiver struct {
	cfg      *Config
	settings receiver.Settings
	consumer consumer.Metrics
	logger   *zap.Logger
	scraper  *natsScraper
	sController *scraperhelper.ScraperController
}

type natsScraper struct {
	cfg      *Config
	logger   *zap.Logger
	registry *prometheus.Registry
}

func newNatsReceiver(
	params receiver.Settings,
	cfg *Config,
	consumer consumer.Metrics,
) receiver.Metrics {
	return &natsReceiver{
		cfg:      cfg,
		settings: params,
		consumer: consumer,
		logger:   params.Logger,
	}
}

func (r *natsReceiver) Start(ctx context.Context, host component.Host) error {
	r.scraper = &natsScraper{
		cfg:      r.cfg,
		logger:   r.logger,
		registry: prometheus.NewRegistry(),
	}

	if err := r.scraper.initCollectors(); err != nil {
		return err
	}

	scraper, err := scraperhelper.NewScraper(
		typeStr.String(),
		r.scraper.Scrape,
		scraperhelper.WithStart(r.scraper.Start),
		scraperhelper.WithShutdown(r.scraper.Shutdown),
	)
	if err != nil {
		return err
	}

	r.sController, err = scraperhelper.NewScraperController(
		scraperhelper.ControllerConfig{
			CollectionInterval: r.cfg.CollectionInterval,
		},
		r.settings,
		r.consumer,
		scraperhelper.AddScraper(scraper),
	)
	if err != nil {
		return err
	}

	return r.sController.Start(ctx, host)
}

func (r *natsReceiver) Shutdown(ctx context.Context) error {
	if r.sController != nil {
		return r.sController.Shutdown(ctx)
	}
	return nil
}

func (s *natsScraper) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (s *natsScraper) Shutdown(ctx context.Context) error {
	return nil
}

func (s *natsScraper) initCollectors() error {
	// Parse server URL
	// The original code supports parsing ID from URL or explicit ID.
	// For now, we assume Endpoint is just the URL and we derive ID or use defaults.
	// Logic from main.go parseServerIDAndURL could be useful, but simpler here:
	url := s.cfg.Endpoint
	id := url // Default ID is URL
	
	// Create CollectedServer
	cs := &collector.CollectedServer{ID: id, URL: url}
	servers := []*collector.CollectedServer{cs}

	// Register collectors based on flags
	register := func(system, endpoint string) {
		c := collector.NewCollector(system, endpoint, "", servers)
		if err := s.registry.Register(c); err != nil {
			s.logger.Warn("Failed to register collector", zap.String("endpoint", endpoint), zap.Error(err))
		}
	}

	if s.cfg.GetVarz {
		register(collector.CoreSystem, "varz")
	}
	if s.cfg.GetConnz {
		register(collector.CoreSystem, "connz")
	}
	if s.cfg.GetConnzDetailed {
		register(collector.CoreSystem, "connz_detailed")
	}
	if s.cfg.GetHealthz {
		register(collector.CoreSystem, "healthz")
	}
	if s.cfg.GetHealthzJsEnabledOnly {
		register(collector.CoreSystem, "healthz_js_enabled_only")
	}
	if s.cfg.GetHealthzJsServerOnly {
		register(collector.CoreSystem, "healthz_js_server_only")
	}
	if s.cfg.GetGatewayz {
		register(collector.CoreSystem, "gatewayz")
	}
	if s.cfg.GetAccstatz {
		register(collector.CoreSystem, "accstatz")
	}
	if s.cfg.GetAccountz {
		register(collector.CoreSystem, "accountz")
	}
	if s.cfg.GetLeafz {
		register(collector.CoreSystem, "leafz")
	}
	if s.cfg.GetRoutez {
		register(collector.CoreSystem, "routez")
	}
	if s.cfg.GetSubz {
		register(collector.CoreSystem, "subz")
	}

	// JetStream
	if s.cfg.GetJszFilter != "" {
		streamMetaKeys := strings.Split(s.cfg.JszSteamMetaKeys, ",")
		consumerMetaKeys := strings.Split(s.cfg.JszConsumerMetaKeys, ",")
		c := collector.NewJszCollector(s.cfg.GetJszFilter, "", servers, streamMetaKeys, consumerMetaKeys)
		if err := s.registry.Register(c); err != nil {
			s.logger.Warn("Failed to register JSZ collector", zap.Error(err))
		}
	}

	return nil
}

func (s *natsScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// Gather metrics from Prometheus registry
	mfs, err := s.registry.Gather()
	if err != nil {
		if len(mfs) == 0 {
			return pmetric.NewMetrics(), err
		}
		s.logger.Warn("Partial success gathering metrics", zap.Error(err))
	}

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	// We can set resource attributes here if we want to associate all metrics with the NATS server.
	// However, since metrics might have "server_id" label, maybe we don't set a global resource ID
	// unless we are sure.
	
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("github.com/nats-io/prometheus-nats-exporter")

	for _, mf := range mfs {
		name := mf.GetName()
		// help := mf.GetHelp()
		
		switch mf.GetType() {
		case dto.MetricType_GAUGE:
			m := sm.Metrics().AppendEmpty()
			m.SetName(name)
			// m.SetDescription(help)
			gauge := m.SetEmptyGauge()
			
			for _, pmetricMetric := range mf.GetMetric() {
				dp := gauge.DataPoints().AppendEmpty()
				dp.SetDoubleValue(pmetricMetric.GetGauge().GetValue())
				dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now())) // or use timestamp if available in pmetricMetric
				
				if pmetricMetric.TimestampMs != nil {
					dp.SetTimestamp(pcommon.Timestamp(uint64(*pmetricMetric.TimestampMs) * 1000000))
				}

				for _, label := range pmetricMetric.GetLabel() {
					dp.Attributes().PutStr(label.GetName(), label.GetValue())
				}
			}
		
		case dto.MetricType_COUNTER:
			m := sm.Metrics().AppendEmpty()
			m.SetName(name)
			// m.SetDescription(help)
			sum := m.SetEmptySum()
			sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
			sum.SetIsMonotonic(true) // Prometheus counters are monotonic

			for _, pmetricMetric := range mf.GetMetric() {
				dp := sum.DataPoints().AppendEmpty()
				dp.SetDoubleValue(pmetricMetric.GetCounter().GetValue())
				dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				if pmetricMetric.TimestampMs != nil {
					dp.SetTimestamp(pcommon.Timestamp(uint64(*pmetricMetric.TimestampMs) * 1000000))
				}

				for _, label := range pmetricMetric.GetLabel() {
					dp.Attributes().PutStr(label.GetName(), label.GetValue())
				}
			}

		default:
			// Handle other types or skip
		}
	}

	return md, nil
}
