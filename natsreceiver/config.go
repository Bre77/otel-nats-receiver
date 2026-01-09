package natsreceiver

import (
	"encoding/json"
	"fmt"
	"time"
)

// MetricFilter configures which metrics to collect from an endpoint.
// It can be unmarshaled from either a boolean or a list of metric names.
type MetricFilter struct {
	Enabled bool     // Whether to collect from this endpoint
	Metrics []string // If non-empty, only collect these metrics (suffix only, prefix stripped)
}

// UnmarshalJSON handles both boolean and array inputs.
func (m *MetricFilter) UnmarshalJSON(data []byte) error {
	// Try boolean first
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		m.Enabled = b
		m.Metrics = nil
		return nil
	}

	// Try array of strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		if len(arr) == 0 {
			return fmt.Errorf("metric filter list cannot be empty; use false to disable")
		}
		m.Enabled = true
		m.Metrics = arr
		return nil
	}

	return fmt.Errorf("metric filter must be a boolean or list of strings")
}

// MarshalJSON serializes the filter.
func (m MetricFilter) MarshalJSON() ([]byte, error) {
	if len(m.Metrics) > 0 {
		return json.Marshal(m.Metrics)
	}
	return json.Marshal(m.Enabled)
}

// UnmarshalYAML handles both boolean and array inputs for YAML config.
func (m *MetricFilter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try boolean first
	var b bool
	if err := unmarshal(&b); err == nil {
		m.Enabled = b
		m.Metrics = nil
		return nil
	}

	// Try array of strings
	var arr []string
	if err := unmarshal(&arr); err == nil {
		if len(arr) == 0 {
			return fmt.Errorf("metric filter list cannot be empty; use false to disable")
		}
		m.Enabled = true
		m.Metrics = arr
		return nil
	}

	return fmt.Errorf("metric filter must be a boolean or list of strings")
}

// ShouldCollect returns true if the given metric suffix should be collected.
func (m *MetricFilter) ShouldCollect(metricSuffix string) bool {
	if !m.Enabled {
		return false
	}
	if len(m.Metrics) == 0 {
		return true // collect all
	}
	for _, allowed := range m.Metrics {
		if allowed == metricSuffix {
			return true
		}
	}
	return false
}

// Config defines configuration for the NATS receiver.
type Config struct {
	// Endpoint is the URL of the NATS server monitoring endpoint (e.g., http://localhost:8222).
	Endpoint string `mapstructure:"endpoint"`

	// CollectionInterval is the interval at which metrics are collected.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// Core server metrics - can be bool or list of metric names to collect
	GetVarz          MetricFilter `mapstructure:"varz"`           // General server stats
	GetConnz         MetricFilter `mapstructure:"connz"`          // Connection info
	GetConnzDetailed MetricFilter `mapstructure:"connz_detailed"` // Detailed connection info
	GetRoutez        MetricFilter `mapstructure:"routez"`         // Route info
	GetSubz          MetricFilter `mapstructure:"subz"`           // Subscription info
	GetLeafz         MetricFilter `mapstructure:"leafz"`          // Leaf node info
	GetGatewayz      MetricFilter `mapstructure:"gatewayz"`       // Gateway info

	// Health check metrics
	GetHealthz              MetricFilter `mapstructure:"healthz"`                 // Basic health status
	GetHealthzJsEnabledOnly MetricFilter `mapstructure:"healthz_js_enabled_only"` // JetStream enabled health
	GetHealthzJsServerOnly  MetricFilter `mapstructure:"healthz_js_server_only"`  // JetStream server health

	// Account metrics
	GetAccstatz MetricFilter `mapstructure:"accstatz"` // Account stats
	GetAccountz MetricFilter `mapstructure:"accountz"` // Account info

	// JetStream metrics
	GetJsz string `mapstructure:"jsz"` // JetStream filter (e.g., "all", "streams", "consumers", or specific stream name)

	// Server identification options
	UseInternalServerID bool `mapstructure:"use_internal_server_id"`   // Use server_id from varz as identifier
	UseServerName       bool `mapstructure:"use_internal_server_name"` // Use server_name from varz as identifier
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint must be specified")
	}
	if c.CollectionInterval <= 0 {
		return fmt.Errorf("collection_interval must be positive")
	}
	return nil
}
