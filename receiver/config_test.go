package natsreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				GetVarz:                 true,
				GetConnz:                true,
				GetConnzDetailed:        true,
				GetRoutez:               true,
				GetSubz:                 true,
				GetLeafz:                true,
				GetGatewayz:             true,
				GetHealthz:              true,
				GetHealthzJsEnabledOnly: true,
				GetHealthzJsServerOnly:  true,
				GetAccstatz:             true,
				GetAccountz:             true,
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
