package natsreceiver

import (
	"fmt"
	"time"
)

// Config defines configuration for the NATS receiver.
type Config struct {
	// Endpoint is the URL of the NATS server monitoring endpoint (e.g., http://localhost:8222).
	Endpoint string `mapstructure:"endpoint"`

	// CollectionInterval is the interval at which metrics are collected.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// Core server metrics
	GetVarz          bool `mapstructure:"varz"`           // General server stats
	GetConnz         bool `mapstructure:"connz"`          // Connection info
	GetConnzDetailed bool `mapstructure:"connz_detailed"` // Detailed connection info
	GetRoutez        bool `mapstructure:"routez"`         // Route info
	GetSubz          bool `mapstructure:"subz"`           // Subscription info
	GetLeafz         bool `mapstructure:"leafz"`          // Leaf node info
	GetGatewayz      bool `mapstructure:"gatewayz"`       // Gateway info

	// Health check metrics
	GetHealthz              bool `mapstructure:"healthz"`                 // Basic health status
	GetHealthzJsEnabledOnly bool `mapstructure:"healthz_js_enabled_only"` // JetStream enabled health
	GetHealthzJsServerOnly  bool `mapstructure:"healthz_js_server_only"`  // JetStream server health

	// Account metrics
	GetAccstatz bool `mapstructure:"accstatz"` // Account stats
	GetAccountz bool `mapstructure:"accountz"` // Account info

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
