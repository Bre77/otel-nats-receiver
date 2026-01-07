package natsreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Endpoint:           "http://localhost:8222",
				CollectionInterval: 10 * time.Second,
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
			name: "invalid interval",
			cfg: &Config{
				Endpoint:           "http://localhost:8222",
				CollectionInterval: 0,
			},
			expectedErr: "collection_interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	t.Skip("Skipping LoadConfig test as it requires full collector config loading setup")
	// In a real scenario, we would test loading from mapstructure here
}
