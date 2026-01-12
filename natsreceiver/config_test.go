package natsreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "valid config with defaults",
			cfg: &Config{
				Endpoint:           "http://localhost:8222",
				CollectionInterval: 10 * time.Second,
			},
			expectedErr: "",
		},
		{
			name: "valid config with all options",
			cfg: &Config{
				Endpoint:                "http://nats-server:8222",
				CollectionInterval:      30 * time.Second,
				GetVarz:                 MetricFilter{Enabled: true},
				GetConnz:                MetricFilter{Enabled: true},
				GetConnzDetailed:        MetricFilter{Enabled: true},
				GetRoutez:               MetricFilter{Enabled: true},
				GetSubz:                 MetricFilter{Enabled: true},
				GetLeafz:                MetricFilter{Enabled: true},
				GetGatewayz:             MetricFilter{Enabled: true},
				GetHealthz:              MetricFilter{Enabled: true},
				GetHealthzJsEnabledOnly: MetricFilter{Enabled: true},
				GetHealthzJsServerOnly:  MetricFilter{Enabled: true},
				GetAccstatz:             MetricFilter{Enabled: true},
				GetAccountz:             MetricFilter{Enabled: true},
				GetJsz:                  "all",
				UseInternalServerID:     true,
				UseServerName:           false,
			},
			expectedErr: "",
		},
		{
			name: "missing endpoint",
			cfg: &Config{
				Endpoint:           "",
				CollectionInterval: 10 * time.Second,
			},
			expectedErr: "endpoint must be specified",
		},
		{
			name: "zero collection interval",
			cfg: &Config{
				Endpoint:           "http://localhost:8222",
				CollectionInterval: 0,
			},
			expectedErr: "collection_interval must be positive",
		},
		{
			name: "negative collection interval",
			cfg: &Config{
				Endpoint:           "http://localhost:8222",
				CollectionInterval: -1 * time.Second,
			},
			expectedErr: "collection_interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricFilterUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected MetricFilter
	}{
		{
			name:     "boolean true",
			input:    map[string]any{"varz": true},
			expected: MetricFilter{Enabled: true},
		},
		{
			name:     "boolean false",
			input:    map[string]any{"varz": false},
			expected: MetricFilter{Enabled: false},
		},
		{
			name:     "list of metrics",
			input:    map[string]any{"varz": []any{"cpu", "mem"}},
			expected: MetricFilter{Enabled: true, Metrics: []string{"cpu", "mem"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add required fields
			tt.input["endpoint"] = "http://localhost:8222"
			tt.input["collection_interval"] = "10s"

			conf := confmap.NewFromStringMap(tt.input)
			cfg := &Config{}
			err := conf.Unmarshal(cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Enabled, cfg.GetVarz.Enabled)
			assert.Equal(t, tt.expected.Metrics, cfg.GetVarz.Metrics)
		})
	}
}

func TestMetricFilterUnmarshalErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expectedErr string
	}{
		{
			name:        "empty list",
			input:       map[string]any{"varz": []any{}},
			expectedErr: "varz: metric filter list cannot be empty; use false to disable",
		},
		{
			name:        "list with non-string",
			input:       map[string]any{"varz": []any{"cpu", 123}},
			expectedErr: "varz: metric filter list must contain only strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.input["endpoint"] = "http://localhost:8222"
			tt.input["collection_interval"] = "10s"

			conf := confmap.NewFromStringMap(tt.input)
			cfg := &Config{}
			err := conf.Unmarshal(cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}
