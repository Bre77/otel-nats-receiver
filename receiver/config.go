package natsreceiver

import (
	"fmt"
	"time"
)

// Config defines configuration for the NATS receiver.
type Config struct {
	// Endpoint is the URL of the NATS server monitoring endpoint.
	Endpoint string `mapstructure:"endpoint"`

	// CollectionInterval is the interval at which metrics are collected.
	CollectionInterval time.Duration `mapstructure:"collection_interval"`

	// Metrics to scrape
	GetVarz                 bool   `mapstructure:"varz"`
	GetConnz                bool   `mapstructure:"connz"`
	GetConnzDetailed        bool   `mapstructure:"connz_detailed"`
	GetHealthz              bool   `mapstructure:"healthz"`
	GetHealthzJsEnabledOnly bool   `mapstructure:"healthz_js_enabled_only"`
	GetHealthzJsServerOnly  bool   `mapstructure:"healthz_js_server_only"`
	GetGatewayz             bool   `mapstructure:"gatewayz"`
	GetAccstatz             bool   `mapstructure:"accstatz"`
	GetAccountz             bool   `mapstructure:"accountz"`
	GetLeafz                bool   `mapstructure:"leafz"`
	GetRoutez               bool   `mapstructure:"routez"`
	GetSubz                 bool   `mapstructure:"subz"`
	GetJszFilter            string `mapstructure:"jsz"`
	JszStreamMetaKeys       string `mapstructure:"jsz_stream_meta_keys"`
	JszConsumerMetaKeys     string `mapstructure:"jsz_consumer_meta_keys"`

	// NATS Server Options
	UseInternalServerID bool `mapstructure:"use_internal_server_id"`
	UseServerName       bool `mapstructure:"use_internal_server_name"`
}

func (c *Config) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint must be specified")
	}
	if c.CollectionInterval <= 0 {
		return fmt.Errorf("collection_interval must be positive")
	}
	return nil
}
