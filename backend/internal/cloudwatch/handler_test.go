package cloudwatch

import (
	"testing"
)

func TestGetTimeRange(t *testing.T) {
	tests := []struct {
		name           string
		rangeParam     string
		expectedPeriod int32
	}{
		{"default 1h", "", 60},
		{"1h", "1h", 60},
		{"6h", "6h", 300},
		{"24h", "24h", 900},
		{"7d", "7d", 3600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, period := getTimeRange(tt.rangeParam)

			if period != tt.expectedPeriod {
				t.Errorf("expected period %d, got %d", tt.expectedPeriod, period)
			}

			if !start.Before(end) {
				t.Errorf("start should be before end")
			}
		})
	}
}

func TestCalculateStats(t *testing.T) {
	tests := []struct {
		name    string
		values  []float64
		wantAvg float64
		wantMin float64
		wantMax float64
		wantSum float64
	}{
		{
			name:    "normal values",
			values:  []float64{10, 20, 30, 40, 50},
			wantAvg: 30,
			wantMin: 10,
			wantMax: 50,
			wantSum: 150,
		},
		{
			name:    "single value",
			values:  []float64{42},
			wantAvg: 42,
			wantMin: 42,
			wantMax: 42,
			wantSum: 42,
		},
		{
			name:    "empty",
			values:  []float64{},
			wantAvg: 0,
			wantMin: 0,
			wantMax: 0,
			wantSum: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateStats(tt.values)

			if stats.Average != tt.wantAvg {
				t.Errorf("average: want %v, got %v", tt.wantAvg, stats.Average)
			}
			if stats.Minimum != tt.wantMin {
				t.Errorf("minimum: want %v, got %v", tt.wantMin, stats.Minimum)
			}
			if stats.Maximum != tt.wantMax {
				t.Errorf("maximum: want %v, got %v", tt.wantMax, stats.Maximum)
			}
			if stats.Sum != tt.wantSum {
				t.Errorf("sum: want %v, got %v", tt.wantSum, stats.Sum)
			}
		})
	}
}

func TestGetMetricQueries(t *testing.T) {
	queries := getMetricQueries()

	// Should have at least Lambda and DynamoDB metrics
	if len(queries) < 5 {
		t.Errorf("expected at least 5 metric queries, got %d", len(queries))
	}

	// Check for expected Lambda metrics
	hasInvocations := false
	hasErrors := false
	for _, q := range queries {
		if q.Namespace == "AWS/Lambda" && q.MetricName == "Invocations" {
			hasInvocations = true
		}
		if q.Namespace == "AWS/Lambda" && q.MetricName == "Errors" {
			hasErrors = true
		}
	}

	if !hasInvocations {
		t.Error("expected Lambda Invocations metric")
	}
	if !hasErrors {
		t.Error("expected Lambda Errors metric")
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Invocations", "invocations"},
		{"4XXError", "n4xxerror"},
		{"some-metric", "some_metric"},
		{"namespace/metric", "namespace_metric"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeID(%q): want %q, got %q", tt.input, tt.expected, result)
			}
		})
	}
}
