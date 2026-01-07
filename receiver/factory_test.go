package natsreceiver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	require.NotNil(t, factory)
	assert.Equal(t, component.MustNewType("nats"), factory.Type())
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	require.NotNil(t, cfg)
	natsConfig, ok := cfg.(*Config)
	require.True(t, ok)

	assert.Equal(t, "http://localhost:8222", natsConfig.Endpoint)
	assert.Equal(t, 60*time.Second, natsConfig.CollectionInterval)
	assert.True(t, natsConfig.GetVarz)
}

func TestCreateMetricsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	receiver, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	require.NotNil(t, receiver)
}

func TestCreateMetricsReceiverWithInvalidConfig(t *testing.T) {
	factory := NewFactory()
	cfg := &Config{
		Endpoint:           "", // Invalid - empty endpoint
		CollectionInterval: 10 * time.Second,
	}

	// The factory doesn't validate on creation, only on Start
	receiver, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		consumertest.NewNop(),
	)

	// Factory creation should succeed
	require.NoError(t, err)
	require.NotNil(t, receiver)

	// But validation should fail
	require.Error(t, cfg.Validate())
}
