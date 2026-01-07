package natsreceiver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestGetServerID(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		responseBody   interface{}
		useServerID    bool
		useServerName  bool
		expectedID     string
		expectError    bool
		errorContains  string
	}{
		{
			name:         "success with server_id",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"server_id":   "NATS-SERVER-123",
				"server_name": "my-nats-server",
			},
			useServerID:   true,
			useServerName: false,
			expectedID:    "NATS-SERVER-123",
			expectError:   false,
		},
		{
			name:         "success with server_name",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"server_id":   "NATS-SERVER-123",
				"server_name": "my-nats-server",
			},
			useServerID:   false,
			useServerName: true,
			expectedID:    "my-nats-server",
			expectError:   false,
		},
		{
			name:         "server_name takes precedence",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"server_id":   "NATS-SERVER-123",
				"server_name": "my-nats-server",
			},
			useServerID:   true,
			useServerName: true,
			expectedID:    "my-nats-server",
			expectError:   false,
		},
		{
			name:          "non-200 response",
			responseCode:  http.StatusServiceUnavailable,
			responseBody:  nil,
			useServerID:   true,
			useServerName: false,
			expectedID:    "",
			expectError:   true,
			errorContains: "unexpected status code 503",
		},
		{
			name:         "empty server_id",
			responseCode: http.StatusOK,
			responseBody: map[string]string{
				"server_id":   "",
				"server_name": "",
			},
			useServerID:   true,
			useServerName: false,
			expectedID:    "",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/varz", r.URL.Path)
				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			scraper := &natsScraper{
				cfg: &Config{
					UseInternalServerID: tt.useServerID,
					UseServerName:       tt.useServerName,
				},
				logger: zap.NewNop(),
			}

			id, err := scraper.getServerID(server.URL)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestGetServerIDConnectionError(t *testing.T) {
	scraper := &natsScraper{
		cfg: &Config{
			UseInternalServerID: true,
		},
		logger: zap.NewNop(),
	}

	// Use an invalid URL that will fail to connect
	_, err := scraper.getServerID("http://localhost:1")
	require.Error(t, err)
}

func TestConvertGauge(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	sm := pmetric.NewScopeMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	value := 42.5
	metrics := []*dto.Metric{
		{
			Gauge: &dto.Gauge{Value: &value},
			Label: []*dto.LabelPair{
				{Name: proto.String("server_id"), Value: proto.String("test-server")},
			},
		},
	}

	scraper.convertGauge(sm, "test_gauge", "Test gauge metric", metrics, now)

	require.Equal(t, 1, sm.Metrics().Len())
	m := sm.Metrics().At(0)
	assert.Equal(t, "test_gauge", m.Name())
	assert.Equal(t, "Test gauge metric", m.Description())
	assert.Equal(t, pmetric.MetricTypeGauge, m.Type())

	dp := m.Gauge().DataPoints().At(0)
	assert.Equal(t, 42.5, dp.DoubleValue())
	assert.Equal(t, "test-server", dp.Attributes().AsRaw()["server_id"])
}

func TestConvertCounter(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	sm := pmetric.NewScopeMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	value := 100.0
	metrics := []*dto.Metric{
		{
			Counter: &dto.Counter{Value: &value},
			Label: []*dto.LabelPair{
				{Name: proto.String("type"), Value: proto.String("messages")},
			},
		},
	}

	scraper.convertCounter(sm, "test_counter", "Test counter metric", metrics, now)

	require.Equal(t, 1, sm.Metrics().Len())
	m := sm.Metrics().At(0)
	assert.Equal(t, "test_counter", m.Name())
	assert.Equal(t, "Test counter metric", m.Description())
	assert.Equal(t, pmetric.MetricTypeSum, m.Type())

	sum := m.Sum()
	assert.True(t, sum.IsMonotonic())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, sum.AggregationTemporality())

	dp := sum.DataPoints().At(0)
	assert.Equal(t, 100.0, dp.DoubleValue())
}

func TestConvertHistogram(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	sm := pmetric.NewScopeMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	sampleCount := uint64(100)
	sampleSum := 500.0
	bucket1Count := uint64(50)
	bucket2Count := uint64(80)
	bucket3Count := uint64(100)
	bound1 := 0.1
	bound2 := 0.5
	bound3 := 1.0

	metrics := []*dto.Metric{
		{
			Histogram: &dto.Histogram{
				SampleCount: &sampleCount,
				SampleSum:   &sampleSum,
				Bucket: []*dto.Bucket{
					{UpperBound: &bound1, CumulativeCount: &bucket1Count},
					{UpperBound: &bound2, CumulativeCount: &bucket2Count},
					{UpperBound: &bound3, CumulativeCount: &bucket3Count},
				},
			},
		},
	}

	scraper.convertHistogram(sm, "test_histogram", "Test histogram metric", metrics, now)

	require.Equal(t, 1, sm.Metrics().Len())
	m := sm.Metrics().At(0)
	assert.Equal(t, "test_histogram", m.Name())
	assert.Equal(t, pmetric.MetricTypeHistogram, m.Type())

	hist := m.Histogram()
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, hist.AggregationTemporality())

	dp := hist.DataPoints().At(0)
	assert.Equal(t, uint64(100), dp.Count())
	assert.Equal(t, 500.0, dp.Sum())
	assert.Equal(t, 3, dp.ExplicitBounds().Len())
}

func TestConvertSummary(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	sm := pmetric.NewScopeMetrics()
	now := pcommon.NewTimestampFromTime(time.Now())

	sampleCount := uint64(1000)
	sampleSum := 5000.0
	q50 := 0.5
	v50 := 10.0
	q99 := 0.99
	v99 := 100.0

	metrics := []*dto.Metric{
		{
			Summary: &dto.Summary{
				SampleCount: &sampleCount,
				SampleSum:   &sampleSum,
				Quantile: []*dto.Quantile{
					{Quantile: &q50, Value: &v50},
					{Quantile: &q99, Value: &v99},
				},
			},
		},
	}

	scraper.convertSummary(sm, "test_summary", "Test summary metric", metrics, now)

	require.Equal(t, 1, sm.Metrics().Len())
	m := sm.Metrics().At(0)
	assert.Equal(t, "test_summary", m.Name())
	assert.Equal(t, pmetric.MetricTypeSummary, m.Type())

	dp := m.Summary().DataPoints().At(0)
	assert.Equal(t, uint64(1000), dp.Count())
	assert.Equal(t, 5000.0, dp.Sum())
	assert.Equal(t, 2, dp.QuantileValues().Len())
}

func TestSetTimestampWithProvidedTimestamp(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	dp := pmetric.NewNumberDataPoint()
	defaultTs := pcommon.NewTimestampFromTime(time.Now())

	// Create a metric with a specific timestamp
	ts := int64(1609459200000) // 2021-01-01 00:00:00 UTC in milliseconds
	pm := &dto.Metric{
		TimestampMs: &ts,
	}

	scraper.setTimestamp(dp, pm, defaultTs)

	// The timestamp should be the one from the metric, converted to nanoseconds
	expectedTs := pcommon.Timestamp(uint64(ts) * 1_000_000)
	assert.Equal(t, expectedTs, dp.Timestamp())
}

func TestSetTimestampWithDefaultTimestamp(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	dp := pmetric.NewNumberDataPoint()
	defaultTs := pcommon.NewTimestampFromTime(time.Now())

	// Create a metric without a timestamp
	pm := &dto.Metric{}

	scraper.setTimestamp(dp, pm, defaultTs)

	// The timestamp should be the default
	assert.Equal(t, defaultTs, dp.Timestamp())
}

func TestSetLabels(t *testing.T) {
	scraper := &natsScraper{
		logger: zap.NewNop(),
	}

	dp := pmetric.NewNumberDataPoint()
	labels := []*dto.LabelPair{
		{Name: proto.String("server_id"), Value: proto.String("test-server")},
		{Name: proto.String("cluster"), Value: proto.String("test-cluster")},
		{Name: proto.String("empty"), Value: proto.String("")},
	}

	scraper.setLabels(dp, labels)

	attrs := dp.Attributes()
	assert.Equal(t, 3, attrs.Len())

	val, ok := attrs.Get("server_id")
	require.True(t, ok)
	assert.Equal(t, "test-server", val.Str())

	val, ok = attrs.Get("cluster")
	require.True(t, ok)
	assert.Equal(t, "test-cluster", val.Str())

	val, ok = attrs.Get("empty")
	require.True(t, ok)
	assert.Equal(t, "", val.Str())
}

func TestScraperStartShutdown(t *testing.T) {
	scraper := &natsScraper{
		cfg:    &Config{},
		logger: zap.NewNop(),
	}

	// Start and Shutdown should be no-ops but not error
	err := scraper.Start(context.Background(), nil)
	assert.NoError(t, err)

	err = scraper.Shutdown(context.Background())
	assert.NoError(t, err)
}
