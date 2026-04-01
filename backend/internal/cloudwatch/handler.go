package cloudwatch

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/DigiStratum/ds-noc-v2/backend/internal/middleware"
)

// CloudWatchClient wraps the AWS CloudWatch client
type CloudWatchClient struct {
	client *cloudwatch.Client
	mu     sync.RWMutex
}

var cwClient *CloudWatchClient

// initClient lazily initializes the CloudWatch client
func getClient(ctx context.Context) (*cloudwatch.Client, error) {
	if cwClient != nil {
		cwClient.mu.RLock()
		c := cwClient.client
		cwClient.mu.RUnlock()
		if c != nil {
			return c, nil
		}
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := cloudwatch.NewFromConfig(cfg)

	cwClient = &CloudWatchClient{}
	cwClient.mu.Lock()
	cwClient.client = client
	cwClient.mu.Unlock()

	return client, nil
}

// MetricDatapoint represents a single data point
type MetricDatapoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

// MetricStatistics contains computed statistics
type MetricStatistics struct {
	Average float64 `json:"average"`
	Maximum float64 `json:"maximum"`
	Minimum float64 `json:"minimum"`
	Sum     float64 `json:"sum"`
}

// MetricResult represents a single metric's data
type MetricResult struct {
	MetricName string            `json:"metricName"`
	Namespace  string            `json:"namespace"`
	Dimensions map[string]string `json:"dimensions"`
	Unit       string            `json:"unit"`
	Datapoints []MetricDatapoint `json:"datapoints"`
	Statistics MetricStatistics  `json:"statistics"`
}

// MetricsResponse is the API response format
type MetricsResponse struct {
	Metrics   []MetricResult `json:"metrics"`
	Period    string         `json:"period"`
	StartTime string         `json:"startTime"`
	EndTime   string         `json:"endTime"`
}

// MetricQuery defines a metric to query
type MetricQuery struct {
	Namespace  string
	MetricName string
	Dimensions map[string]string
	Stat       string
	Unit       types.StandardUnit
}

// getTimeRange parses the range parameter and returns start/end times
func getTimeRange(rangeParam string) (time.Time, time.Time, int32) {
	now := time.Now().UTC()
	var start time.Time
	var period int32 // seconds

	switch rangeParam {
	case "6h":
		start = now.Add(-6 * time.Hour)
		period = 300 // 5 minutes
	case "24h":
		start = now.Add(-24 * time.Hour)
		period = 900 // 15 minutes
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
		period = 3600 // 1 hour
	default: // 1h
		start = now.Add(-1 * time.Hour)
		period = 60 // 1 minute
	}

	return start, now, period
}

// getMetricQueries returns the list of metrics to query based on environment
func getMetricQueries() []MetricQuery {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "ds-noc-v2"
	}

	lambdaName := os.Getenv("LAMBDA_FUNCTION_NAME")
	if lambdaName == "" {
		lambdaName = appName + "-api"
	}

	apiId := os.Getenv("API_GATEWAY_ID")
	tableName := os.Getenv("DYNAMODB_TABLE")
	if tableName == "" {
		tableName = appName
	}

	queries := []MetricQuery{
		// Lambda metrics
		{
			Namespace:  "AWS/Lambda",
			MetricName: "Invocations",
			Dimensions: map[string]string{"FunctionName": lambdaName},
			Stat:       "Sum",
			Unit:       types.StandardUnitCount,
		},
		{
			Namespace:  "AWS/Lambda",
			MetricName: "Errors",
			Dimensions: map[string]string{"FunctionName": lambdaName},
			Stat:       "Sum",
			Unit:       types.StandardUnitCount,
		},
		{
			Namespace:  "AWS/Lambda",
			MetricName: "Duration",
			Dimensions: map[string]string{"FunctionName": lambdaName},
			Stat:       "Average",
			Unit:       types.StandardUnitMilliseconds,
		},
		{
			Namespace:  "AWS/Lambda",
			MetricName: "Throttles",
			Dimensions: map[string]string{"FunctionName": lambdaName},
			Stat:       "Sum",
			Unit:       types.StandardUnitCount,
		},
		{
			Namespace:  "AWS/Lambda",
			MetricName: "ConcurrentExecutions",
			Dimensions: map[string]string{"FunctionName": lambdaName},
			Stat:       "Maximum",
			Unit:       types.StandardUnitCount,
		},
	}

	// Add API Gateway metrics if API ID is configured
	if apiId != "" {
		queries = append(queries,
			MetricQuery{
				Namespace:  "AWS/ApiGateway",
				MetricName: "Count",
				Dimensions: map[string]string{"ApiId": apiId},
				Stat:       "Sum",
				Unit:       types.StandardUnitCount,
			},
			MetricQuery{
				Namespace:  "AWS/ApiGateway",
				MetricName: "4XXError",
				Dimensions: map[string]string{"ApiId": apiId},
				Stat:       "Sum",
				Unit:       types.StandardUnitCount,
			},
			MetricQuery{
				Namespace:  "AWS/ApiGateway",
				MetricName: "5XXError",
				Dimensions: map[string]string{"ApiId": apiId},
				Stat:       "Sum",
				Unit:       types.StandardUnitCount,
			},
			MetricQuery{
				Namespace:  "AWS/ApiGateway",
				MetricName: "Latency",
				Dimensions: map[string]string{"ApiId": apiId},
				Stat:       "Average",
				Unit:       types.StandardUnitMilliseconds,
			},
		)
	}

	// Add DynamoDB metrics
	queries = append(queries,
		MetricQuery{
			Namespace:  "AWS/DynamoDB",
			MetricName: "ConsumedReadCapacityUnits",
			Dimensions: map[string]string{"TableName": tableName},
			Stat:       "Sum",
			Unit:       types.StandardUnitCount,
		},
		MetricQuery{
			Namespace:  "AWS/DynamoDB",
			MetricName: "ConsumedWriteCapacityUnits",
			Dimensions: map[string]string{"TableName": tableName},
			Stat:       "Sum",
			Unit:       types.StandardUnitCount,
		},
	)

	return queries
}

// Handler handles GET /api/cloudwatch/metrics
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only GET is supported")
		return
	}

	ctx := r.Context()
	logger := middleware.LoggerWithCorrelation(ctx)

	// Set txnlog data type for transaction logging
	middleware.SetTxnLogDataType(ctx, "CloudWatchMetric")

	// Parse query parameters
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "1h"
	}

	// Validate range
	validRanges := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true}
	if !validRanges[rangeParam] {
		WriteError(w, r, http.StatusBadRequest, "INVALID_RANGE", "range must be one of: 1h, 6h, 24h, 7d")
		return
	}

	// Get CloudWatch client
	client, err := getClient(ctx)
	if err != nil {
		logger.Error("failed to create CloudWatch client", "error", err)
		WriteError(w, r, http.StatusInternalServerError, "CLIENT_ERROR", "Failed to initialize CloudWatch client")
		return
	}

	// Calculate time range and period
	startTime, endTime, period := getTimeRange(rangeParam)
	queries := getMetricQueries()

	// Build GetMetricData input
	metricDataQueries := make([]types.MetricDataQuery, len(queries))
	for i, q := range queries {
		dims := make([]types.Dimension, 0, len(q.Dimensions))
		for name, value := range q.Dimensions {
			dims = append(dims, types.Dimension{
				Name:  aws.String(name),
				Value: aws.String(value),
			})
		}

		metricDataQueries[i] = types.MetricDataQuery{
			Id: aws.String("m" + strconv.Itoa(i) + "_" + sanitizeID(q.MetricName)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String(q.Namespace),
					MetricName: aws.String(q.MetricName),
					Dimensions: dims,
				},
				Period: aws.Int32(period),
				Stat:   aws.String(q.Stat),
				Unit:   q.Unit,
			},
			ReturnData: aws.Bool(true),
		}
	}

	// Fetch metrics from CloudWatch
	logger.Debug("fetching CloudWatch metrics",
		"range", rangeParam,
		"start", startTime.Format(time.RFC3339),
		"end", endTime.Format(time.RFC3339),
		"queries", len(queries))

	output, err := client.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: metricDataQueries,
	})
	if err != nil {
		logger.Error("CloudWatch GetMetricData failed", "error", err)
		WriteError(w, r, http.StatusInternalServerError, "CLOUDWATCH_ERROR", "Failed to fetch metrics from CloudWatch")
		return
	}

	// Map results back to our response format
	idToQuery := make(map[string]MetricQuery)
	for i, q := range queries {
		id := "m" + strconv.Itoa(i) + "_" + sanitizeID(q.MetricName)
		idToQuery[id] = queries[i]
	}

	results := make([]MetricResult, 0, len(output.MetricDataResults))
	for _, mdr := range output.MetricDataResults {
		if mdr.Id == nil {
			continue
		}

		query, ok := idToQuery[*mdr.Id]
		if !ok {
			continue
		}

		// Build datapoints
		datapoints := make([]MetricDatapoint, len(mdr.Timestamps))
		for j, ts := range mdr.Timestamps {
			value := 0.0
			if j < len(mdr.Values) {
				value = mdr.Values[j]
			}
			datapoints[j] = MetricDatapoint{
				Timestamp: ts.Format(time.RFC3339),
				Value:     value,
				Unit:      string(query.Unit),
			}
		}

		// Sort datapoints by timestamp
		sort.Slice(datapoints, func(i, j int) bool {
			return datapoints[i].Timestamp < datapoints[j].Timestamp
		})

		// Calculate statistics
		stats := calculateStats(mdr.Values)

		results = append(results, MetricResult{
			MetricName: query.MetricName,
			Namespace:  query.Namespace,
			Dimensions: query.Dimensions,
			Unit:       string(query.Unit),
			Datapoints: datapoints,
			Statistics: stats,
		})
	}

	// Sort results by namespace then metric name
	sort.Slice(results, func(i, j int) bool {
		if results[i].Namespace != results[j].Namespace {
			return results[i].Namespace < results[j].Namespace
		}
		return results[i].MetricName < results[j].MetricName
	})

	logger.Info("CloudWatch metrics fetched", "count", len(results))

	WriteJSON(w, http.StatusOK, MetricsResponse{
		Metrics:   results,
		Period:    rangeParam,
		StartTime: startTime.Format(time.RFC3339),
		EndTime:   endTime.Format(time.RFC3339),
	})
}

// calculateStats computes statistics from a slice of values
func calculateStats(values []float64) MetricStatistics {
	if len(values) == 0 {
		return MetricStatistics{}
	}

	var sum, min, max float64
	min = values[0]
	max = values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	return MetricStatistics{
		Average: sum / float64(len(values)),
		Maximum: max,
		Minimum: min,
		Sum:     sum,
	}
}

// ErrorResponse represents a standard error
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	correlationID := middleware.GetCorrelationID(r.Context())
	WriteJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: correlationID,
		},
	})
}

// sanitizeID converts a metric name to a valid CloudWatch query ID
// Pattern: ^[a-z][a-zA-Z0-9_]*$
func sanitizeID(s string) string {
	result := strings.ToLower(s)
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "-", "_")
	// If starts with number, prefix with underscore
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "n" + result
	}
	return result
}
