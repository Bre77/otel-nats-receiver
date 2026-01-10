package natsreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/prometheus-nats-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
	"go.uber.org/zap"
)

// varzResponse contains the NATS server information from /varz endpoint
type varzResponse struct {
	ServerID       string `json:"server_id"`
	ServerName     string `json:"server_name"`
	Version        string `json:"version"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Cluster        *struct {
		Name string `json:"name"`
	} `json:"cluster,omitempty"`
	JetStream      bool      `json:"jetstream"`
	MaxConnections int       `json:"max_connections"`
	MaxPayload     int       `json:"max_payload"`
	ConfigLoadTime time.Time `json:"config_load_time"`
}

type natsReceiver struct {
	cfg                *Config
	settings           receiver.Settings
	metricsConsumer    consumer.Metrics
	logsConsumer       consumer.Logs
	logger             *zap.Logger
	scraper            *natsScraper
	sController        receiver.Metrics
	lastConfigLoadTime time.Time
	configMu           sync.Mutex
}

type natsScraper struct {
	cfg      *Config
	logger   *zap.Logger
	registry *prometheus.Registry
	receiver *natsReceiver
}

func newNatsReceiver(
	params receiver.Settings,
	cfg *Config,
	metricsConsumer consumer.Metrics,
	logsConsumer consumer.Logs,
) *natsReceiver {
	return &natsReceiver{
		cfg:             cfg,
		settings:        params,
		metricsConsumer: metricsConsumer,
		logsConsumer:    logsConsumer,
		logger:          params.Logger,
	}
}

func (r *natsReceiver) Start(ctx context.Context, host component.Host) error {
	r.scraper = &natsScraper{
		cfg:      r.cfg,
		logger:   r.logger,
		registry: prometheus.NewRegistry(),
		receiver: r,
	}

	if err := r.scraper.initCollectors(); err != nil {
		return err
	}

	// Emit startup log if enabled and logs consumer is available
	if r.cfg.StartupLog && r.logsConsumer != nil {
		varz, err := r.fetchVarz()
		if err != nil {
			r.logger.Warn("Failed to fetch varz for startup log", zap.Error(err))
		} else {
			r.configMu.Lock()
			r.lastConfigLoadTime = varz.ConfigLoadTime
			r.configMu.Unlock()
			if err := r.emitLog(ctx, varz, "NATS server connected"); err != nil {
				r.logger.Warn("Failed to emit startup log", zap.Error(err))
			}
		}
	}

	// Only start metrics controller if metrics consumer is available
	if r.metricsConsumer != nil {
		scrp, err := scraper.NewMetrics(
			r.scraper.Scrape,
			scraper.WithStart(r.scraper.Start),
			scraper.WithShutdown(r.scraper.Shutdown),
		)
		if err != nil {
			return err
		}

		r.sController, err = scraperhelper.NewMetricsController(
			&scraperhelper.ControllerConfig{
				CollectionInterval: r.cfg.CollectionInterval,
			},
			r.settings,
			r.metricsConsumer,
			scraperhelper.AddScraper(typeStr, scrp),
		)
		if err != nil {
			return err
		}

		return r.sController.Start(ctx, host)
	}

	return nil
}

func (r *natsReceiver) Shutdown(ctx context.Context) error {
	if r.sController != nil {
		return r.sController.Shutdown(ctx)
	}
	return nil
}

// fetchVarz retrieves the full varz response from the NATS server
func (r *natsReceiver) fetchVarz() (*varzResponse, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	url := strings.TrimRight(r.cfg.Endpoint, "/") + "/varz"
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	var v varzResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// emitLog creates and sends an OpenTelemetry log record with NATS server information
func (r *natsReceiver) emitLog(ctx context.Context, varz *varzResponse, message string) error {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()

	// Set resource attributes (OTel semantic conventions)
	res := rl.Resource()
	res.Attributes().PutStr("service.name", "nats")
	res.Attributes().PutStr("service.instance.id", varz.ServerID)
	res.Attributes().PutStr("service.version", varz.Version)
	res.Attributes().PutStr("host.name", varz.ServerName)

	sl := rl.ScopeLogs().AppendEmpty()
	sl.Scope().SetName("github.com/Bre77/otel-nats-receiver")
	sl.Scope().SetVersion("0.1.0")

	lr := sl.LogRecords().AppendEmpty()
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")
	lr.Body().SetStr(message)

	// Set log attributes
	attrs := lr.Attributes()
	attrs.PutStr("host", varz.Host)
	attrs.PutInt("port", int64(varz.Port))
	attrs.PutBool("jetstream.enabled", varz.JetStream)
	attrs.PutInt("max_connections", int64(varz.MaxConnections))
	attrs.PutInt("max_payload", int64(varz.MaxPayload))
	attrs.PutStr("config_load_time", varz.ConfigLoadTime.Format(time.RFC3339))
	if varz.Cluster != nil && varz.Cluster.Name != "" {
		attrs.PutStr("cluster.name", varz.Cluster.Name)
	}

	return r.logsConsumer.ConsumeLogs(ctx, ld)
}

func (s *natsScraper) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (s *natsScraper) Shutdown(ctx context.Context) error {
	return nil
}

// checkConfigReload checks if the NATS server config has been reloaded and emits a log if so
func (s *natsScraper) checkConfigReload(ctx context.Context) {
	varz, err := s.receiver.fetchVarz()
	if err != nil {
		s.logger.Debug("Failed to fetch varz for config reload check", zap.Error(err))
		return
	}

	s.receiver.configMu.Lock()
	lastTime := s.receiver.lastConfigLoadTime
	s.receiver.configMu.Unlock()

	// Only emit log if we have a previous time and it has changed
	if !lastTime.IsZero() && !varz.ConfigLoadTime.Equal(lastTime) {
		s.receiver.configMu.Lock()
		s.receiver.lastConfigLoadTime = varz.ConfigLoadTime
		s.receiver.configMu.Unlock()

		if err := s.receiver.emitLog(ctx, varz, "NATS server configuration reloaded"); err != nil {
			s.logger.Warn("Failed to emit config reload log", zap.Error(err))
		}
	} else if lastTime.IsZero() {
		// Initialize the last config load time if not set (e.g., startup_log was disabled)
		s.receiver.configMu.Lock()
		s.receiver.lastConfigLoadTime = varz.ConfigLoadTime
		s.receiver.configMu.Unlock()
	}
}

func (s *natsScraper) initCollectors() error {
	url := s.cfg.Endpoint
	id := url

	if s.cfg.UseInternalServerID || s.cfg.UseServerName {
		fetchedID, err := s.getServerID(url)
		if err != nil {
			s.logger.Warn("Failed to fetch server ID/Name from varz, using URL as ID", zap.Error(err))
		} else {
			id = fetchedID
		}
	}

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

	if s.cfg.GetVarz.Enabled {
		register(collector.CoreSystem, "varz")
	}
	if s.cfg.GetConnz.Enabled {
		register(collector.CoreSystem, "connz")
	}
	if s.cfg.GetConnzDetailed.Enabled {
		register(collector.CoreSystem, "connz_detailed")
	}
	if s.cfg.GetHealthz.Enabled {
		register(collector.CoreSystem, "healthz")
	}
	if s.cfg.GetHealthzJsEnabledOnly.Enabled {
		register(collector.CoreSystem, "healthz_js_enabled_only")
	}
	if s.cfg.GetHealthzJsServerOnly.Enabled {
		register(collector.CoreSystem, "healthz_js_server_only")
	}
	if s.cfg.GetGatewayz.Enabled {
		register(collector.CoreSystem, "gatewayz")
	}
	if s.cfg.GetAccstatz.Enabled {
		register(collector.CoreSystem, "accstatz")
	}
	if s.cfg.GetAccountz.Enabled {
		register(collector.CoreSystem, "accountz")
	}
	if s.cfg.GetLeafz.Enabled {
		register(collector.CoreSystem, "leafz")
	}
	if s.cfg.GetRoutez.Enabled {
		register(collector.CoreSystem, "routez")
	}
	if s.cfg.GetSubz.Enabled {
		register(collector.CoreSystem, "subz")
	}

	// JetStream
	if s.cfg.GetJsz != "" {
		c := collector.NewCollector(collector.JetStreamSystem, s.cfg.GetJsz, "", servers)
		if err := s.registry.Register(c); err != nil {
			s.logger.Warn("Failed to register JetStream collector", zap.Error(err))
		}
	}

	return nil
}

func (s *natsScraper) getServerID(baseURL string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	// Append /varz to base URL. Handle potential trailing slash.
	url := strings.TrimRight(baseURL, "/") + "/varz"
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d from %s", resp.StatusCode, url)
	}

	var v struct {
		ServerID   string `json:"server_id"`
		ServerName string `json:"server_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}

	if s.cfg.UseServerName && v.ServerName != "" {
		return v.ServerName, nil
	}
	if s.cfg.UseInternalServerID && v.ServerID != "" {
		return v.ServerID, nil
	}
	return "", nil
}

// metricPrefixes maps metric name prefixes to their config field names and OTel prefix
var metricPrefixes = []struct {
	oldPrefix string // e.g., "gnatsd_varz_"
	newPrefix string // e.g., "nats.varz."
	field     string // config field name
}{
	{"gnatsd_varz_", "nats.varz.", "varz"},
	{"gnatsd_connz_", "nats.connz.", "connz"},
	{"gnatsd_routez_", "nats.routez.", "routez"},
	{"gnatsd_subz_", "nats.subz.", "subz"},
	{"gnatsd_leafz_", "nats.leafz.", "leafz"},
	{"gnatsd_gatewayz_", "nats.gatewayz.", "gatewayz"},
	{"gnatsd_healthz_", "nats.healthz.", "healthz"},
	{"gnatsd_accstatz_", "nats.accstatz.", "accstatz"},
	{"gnatsd_accountz_", "nats.accountz.", "accountz"},
	{"gnatsd_jsz_", "nats.jsz.", "jsz"},
}

// shouldCollectMetric checks if a metric should be collected based on its name and the config filters.
func (s *natsScraper) shouldCollectMetric(name string) bool {
	for _, p := range metricPrefixes {
		if strings.HasPrefix(name, p.oldPrefix) {
			suffix := strings.TrimPrefix(name, p.oldPrefix)
			filter := s.getFilterForField(p.field)
			if filter == nil {
				return true // no filter, collect all
			}
			return filter.ShouldCollect(suffix)
		}
	}
	return true // unknown prefix, collect by default
}

// transformMetricName converts legacy gnatsd_ prefixed names to OTel-compliant nats. names.
// e.g., "gnatsd_varz_cpu" -> "nats.varz.cpu"
func transformMetricName(name string) string {
	for _, p := range metricPrefixes {
		if strings.HasPrefix(name, p.oldPrefix) {
			suffix := strings.TrimPrefix(name, p.oldPrefix)
			return p.newPrefix + suffix
		}
	}
	return name // return unchanged if no prefix matches
}

// getFilterForField returns the MetricFilter for a given field name.
func (s *natsScraper) getFilterForField(field string) *MetricFilter {
	switch field {
	case "varz":
		return &s.cfg.GetVarz
	case "connz":
		return &s.cfg.GetConnz
	case "routez":
		return &s.cfg.GetRoutez
	case "subz":
		return &s.cfg.GetSubz
	case "leafz":
		return &s.cfg.GetLeafz
	case "gatewayz":
		return &s.cfg.GetGatewayz
	case "healthz":
		return &s.cfg.GetHealthz
	case "accstatz":
		return &s.cfg.GetAccstatz
	case "accountz":
		return &s.cfg.GetAccountz
	default:
		return nil
	}
}

func (s *natsScraper) Scrape(ctx context.Context) (pmetric.Metrics, error) {
	// Check for config reload if enabled
	if s.cfg.ConfigLog && s.receiver != nil && s.receiver.logsConsumer != nil {
		s.checkConfigReload(ctx)
	}

	mfs, err := s.registry.Gather()
	if err != nil {
		if len(mfs) == 0 {
			return pmetric.NewMetrics(), err
		}
		s.logger.Warn("Partial success gathering metrics", zap.Error(err))
	}

	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "nats")

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("github.com/Bre77/otel-nats-receiver")
	sm.Scope().SetVersion("0.1.0")

	now := pcommon.NewTimestampFromTime(time.Now())

	for _, mf := range mfs {
		originalName := mf.GetName()

		// Check if this metric should be collected based on filters
		if !s.shouldCollectMetric(originalName) {
			continue
		}

		// Transform to OTel-compliant name (e.g., gnatsd_varz_cpu -> nats.varz.cpu)
		name := transformMetricName(originalName)
		help := mf.GetHelp()

		switch mf.GetType() {
		case dto.MetricType_GAUGE:
			s.convertGauge(sm, name, help, mf.GetMetric(), now)

		case dto.MetricType_COUNTER:
			s.convertCounter(sm, name, help, mf.GetMetric(), now)

		case dto.MetricType_HISTOGRAM:
			s.convertHistogram(sm, name, help, mf.GetMetric(), now)

		case dto.MetricType_SUMMARY:
			s.convertSummary(sm, name, help, mf.GetMetric(), now)

		case dto.MetricType_UNTYPED:
			// Treat untyped as gauge
			s.convertGauge(sm, name, help, mf.GetMetric(), now)

		default:
			s.logger.Debug("Skipping unsupported metric type", zap.String("name", name), zap.String("type", mf.GetType().String()))
		}
	}

	return md, nil
}

func (s *natsScraper) convertGauge(sm pmetric.ScopeMetrics, name, description string, metrics []*dto.Metric, defaultTimestamp pcommon.Timestamp) {
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	m.SetDescription(description)
	gauge := m.SetEmptyGauge()

	for _, pm := range metrics {
		dp := gauge.DataPoints().AppendEmpty()
		dp.SetDoubleValue(pm.GetGauge().GetValue())
		s.setTimestamp(dp, pm, defaultTimestamp)
		s.setLabels(dp, pm.GetLabel())
	}
}

func (s *natsScraper) convertCounter(sm pmetric.ScopeMetrics, name, description string, metrics []*dto.Metric, defaultTimestamp pcommon.Timestamp) {
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	m.SetDescription(description)
	sum := m.SetEmptySum()
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	sum.SetIsMonotonic(true)

	for _, pm := range metrics {
		dp := sum.DataPoints().AppendEmpty()
		dp.SetDoubleValue(pm.GetCounter().GetValue())
		s.setTimestamp(dp, pm, defaultTimestamp)
		s.setLabels(dp, pm.GetLabel())
	}
}

func (s *natsScraper) convertHistogram(sm pmetric.ScopeMetrics, name, description string, metrics []*dto.Metric, defaultTimestamp pcommon.Timestamp) {
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	m.SetDescription(description)
	hist := m.SetEmptyHistogram()
	hist.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)

	for _, pm := range metrics {
		h := pm.GetHistogram()
		dp := hist.DataPoints().AppendEmpty()
		dp.SetCount(h.GetSampleCount())
		dp.SetSum(h.GetSampleSum())
		s.setTimestamp(dp, pm, defaultTimestamp)
		s.setLabels(dp, pm.GetLabel())

		// Convert buckets
		buckets := h.GetBucket()
		dp.ExplicitBounds().EnsureCapacity(len(buckets))
		dp.BucketCounts().EnsureCapacity(len(buckets) + 1)

		var prevCount uint64
		for _, b := range buckets {
			dp.ExplicitBounds().Append(b.GetUpperBound())
			// Convert cumulative to delta counts
			dp.BucketCounts().Append(b.GetCumulativeCount() - prevCount)
			prevCount = b.GetCumulativeCount()
		}
	}
}

func (s *natsScraper) convertSummary(sm pmetric.ScopeMetrics, name, description string, metrics []*dto.Metric, defaultTimestamp pcommon.Timestamp) {
	m := sm.Metrics().AppendEmpty()
	m.SetName(name)
	m.SetDescription(description)
	summary := m.SetEmptySummary()

	for _, pm := range metrics {
		su := pm.GetSummary()
		dp := summary.DataPoints().AppendEmpty()
		dp.SetCount(su.GetSampleCount())
		dp.SetSum(su.GetSampleSum())
		s.setTimestamp(dp, pm, defaultTimestamp)
		s.setLabels(dp, pm.GetLabel())

		// Convert quantiles
		for _, q := range su.GetQuantile() {
			qv := dp.QuantileValues().AppendEmpty()
			qv.SetQuantile(q.GetQuantile())
			qv.SetValue(q.GetValue())
		}
	}
}

// setTimestamp sets the timestamp on a data point, using the metric's timestamp if available
func (s *natsScraper) setTimestamp(dp interface{ SetTimestamp(pcommon.Timestamp) }, pm *dto.Metric, defaultTimestamp pcommon.Timestamp) {
	if pm.TimestampMs != nil {
		dp.SetTimestamp(pcommon.Timestamp(uint64(*pm.TimestampMs) * 1_000_000))
	} else {
		dp.SetTimestamp(defaultTimestamp)
	}
}

// setLabels copies Prometheus labels to OTel attributes
func (s *natsScraper) setLabels(dp interface{ Attributes() pcommon.Map }, labels []*dto.LabelPair) {
	for _, label := range labels {
		dp.Attributes().PutStr(label.GetName(), label.GetValue())
	}
}
