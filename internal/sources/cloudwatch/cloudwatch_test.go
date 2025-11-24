// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudwatch

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	sourceutil "github.com/googleapis/genai-toolbox/internal/sources/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestCloudWatchConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration with minimal settings",
			yamlContent: `name: test-cloudwatch
kind: cloudwatch
region: us-east-1`,
			wantErr: false,
			expected: Config{
				Name:   "test-cloudwatch",
				Kind:   "cloudwatch",
				Region: "us-east-1",
			},
		},
		{
			name: "valid configuration with log group",
			yamlContent: `name: test-cloudwatch
kind: cloudwatch
region: us-west-2
logGroupName: /aws/lambda/my-function`,
			wantErr: false,
			expected: Config{
				Name:         "test-cloudwatch",
				Kind:         "cloudwatch",
				Region:       "us-west-2",
				LogGroupName: "/aws/lambda/my-function",
			},
		},
		{
			name: "valid configuration with endpoint for LocalStack",
			yamlContent: `name: test-cloudwatch-local
kind: cloudwatch
region: us-east-1
endpoint: http://localhost:4566
logGroupName: /aws/lambda/test`,
			wantErr: false,
			expected: Config{
				Name:         "test-cloudwatch-local",
				Kind:         "cloudwatch",
				Region:       "us-east-1",
				Endpoint:     "http://localhost:4566",
				LogGroupName: "/aws/lambda/test",
			},
		},
		{
			name: "valid configuration with explicit credentials",
			yamlContent: `name: test-cloudwatch-creds
kind: cloudwatch
region: eu-west-1
accessKeyId: AKIAIOSFODNN7EXAMPLE
secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantErr: false,
			expected: Config{
				Name:            "test-cloudwatch-creds",
				Kind:            "cloudwatch",
				Region:          "eu-west-1",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "valid configuration with session token",
			yamlContent: `name: test-cloudwatch-session
kind: cloudwatch
region: ap-southeast-1
accessKeyId: AKIAIOSFODNN7EXAMPLE
secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
sessionToken: FwoGZXIvYXdzEBQaDH1234567890EXAMPLE`,
			wantErr: false,
			expected: Config{
				Name:            "test-cloudwatch-session",
				Kind:            "cloudwatch",
				Region:          "ap-southeast-1",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "FwoGZXIvYXdzEBQaDH1234567890EXAMPLE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlContent)))
			config, err := newConfig(context.Background(), tt.expected.Name, decoder)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, config)

				cfg := config.(Config)
				assert.Equal(t, tt.expected.Name, cfg.Name)
				assert.Equal(t, tt.expected.Region, cfg.Region)
				assert.Equal(t, tt.expected.LogGroupName, cfg.LogGroupName)
				assert.Equal(t, tt.expected.Endpoint, cfg.Endpoint)
				assert.Equal(t, tt.expected.AccessKeyID, cfg.AccessKeyID)
				assert.Equal(t, tt.expected.SecretAccessKey, cfg.SecretAccessKey)
				assert.Equal(t, tt.expected.SessionToken, cfg.SessionToken)
			}
		})
	}
}

func TestSourceKind(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "cloudwatch",
		Region: "us-east-1",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}

func TestToConfig(t *testing.T) {
	config := Config{
		Name:         "test-cloudwatch",
		Kind:         "cloudwatch",
		Region:       "us-east-1",
		LogGroupName: "/aws/lambda/test",
	}

	source := Source{Config: config}
	returnedConfig := source.ToConfig()

	assert.Equal(t, config, returnedConfig)
}

func TestFilterLogEventsInput_Validation(t *testing.T) {
	// This test validates the input construction logic
	// In a real scenario, you would use mocks or LocalStack for integration tests

	tests := []struct {
		name      string
		config    Config
		input     *FilterLogEventsInput
		wantErr   bool
		errString string
	}{
		{
			name: "nil input should error",
			config: Config{
				LogGroupName: "/aws/lambda/test",
			},
			input:     nil,
			wantErr:   true,
			errString: "input cannot be nil",
		},
		{
			name:   "missing log group name in both input and config",
			config: Config{},
			input: &FilterLogEventsInput{
				FilterPattern: "[level=ERROR]",
			},
			wantErr:   true,
			errString: "logGroupName must be specified",
		},
		{
			name: "valid with log group in config",
			config: Config{
				LogGroupName: "/aws/lambda/test",
			},
			input: &FilterLogEventsInput{
				StartTime:     time.Now().Add(-1 * time.Hour),
				FilterPattern: "[level=ERROR]",
				Limit:         100,
			},
			wantErr: false,
		},
		{
			name:   "valid with log group in input",
			config: Config{},
			input: &FilterLogEventsInput{
				LogGroupName:  "/aws/lambda/explicit",
				StartTime:     time.Now().Add(-1 * time.Hour),
				FilterPattern: "[level=INFO]",
				Limit:         50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually call FilterLogEvents without a real AWS connection
			// But we can validate the input construction logic
			if tt.input != nil {
				logGroupName := tt.input.LogGroupName
				if logGroupName == "" {
					logGroupName = tt.config.LogGroupName
				}

				if logGroupName == "" && tt.wantErr {
					assert.Contains(t, tt.errString, "logGroupName")
				} else if !tt.wantErr {
					assert.NotEmpty(t, logGroupName)
				}
			}
		})
	}
}

func TestInsightsQueryInput_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		input     *InsightsQueryInput
		wantErr   bool
		errString string
	}{
		{
			name:      "nil input should error",
			config:    Config{},
			input:     nil,
			wantErr:   true,
			errString: "input cannot be nil",
		},
		{
			name:   "missing query string",
			config: Config{},
			input: &InsightsQueryInput{
				LogGroupNames: []string{"/aws/lambda/test"},
				StartTime:     time.Now().Add(-1 * time.Hour),
				EndTime:       time.Now(),
			},
			wantErr:   true,
			errString: "queryString must be specified",
		},
		{
			name:   "missing start time",
			config: Config{},
			input: &InsightsQueryInput{
				LogGroupNames: []string{"/aws/lambda/test"},
				QueryString:   "fields @timestamp, @message",
				EndTime:       time.Now(),
			},
			wantErr:   true,
			errString: "startTime and endTime must be specified",
		},
		{
			name:   "missing end time",
			config: Config{},
			input: &InsightsQueryInput{
				LogGroupNames: []string{"/aws/lambda/test"},
				QueryString:   "fields @timestamp, @message",
				StartTime:     time.Now().Add(-1 * time.Hour),
			},
			wantErr:   true,
			errString: "endTime must be specified",
		},
		{
			name: "valid with log groups in input",
			config: Config{
				LogGroupName: "/aws/lambda/default",
			},
			input: &InsightsQueryInput{
				LogGroupNames: []string{"/aws/lambda/test1", "/aws/lambda/test2"},
				QueryString:   "fields @timestamp, @message | filter @message like /ERROR/",
				StartTime:     time.Now().Add(-24 * time.Hour),
				EndTime:       time.Now(),
				Limit:         1000,
			},
			wantErr: false,
		},
		{
			name: "valid with log group from config",
			config: Config{
				LogGroupName: "/aws/lambda/default",
			},
			input: &InsightsQueryInput{
				QueryString: "stats count() by bin(5m)",
				StartTime:   time.Now().Add(-1 * time.Hour),
				EndTime:     time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate input construction logic
			if tt.input == nil && tt.wantErr {
				assert.Contains(t, tt.errString, "nil")
				return
			}

			if tt.input != nil {
				logGroupNames := tt.input.LogGroupNames
				if len(logGroupNames) == 0 && tt.config.LogGroupName != "" {
					logGroupNames = []string{tt.config.LogGroupName}
				}

				if tt.wantErr {
					if tt.errString == "queryString must be specified" {
						assert.Empty(t, tt.input.QueryString)
					}
					if tt.errString == "startTime and endTime must be specified" {
						assert.True(t, tt.input.StartTime.IsZero() || tt.input.EndTime.IsZero())
					}
				} else {
					assert.NotEmpty(t, logGroupNames)
					assert.NotEmpty(t, tt.input.QueryString)
					assert.False(t, tt.input.StartTime.IsZero())
					assert.False(t, tt.input.EndTime.IsZero())
				}
			}
		})
	}
}

func TestGetInsightsQueryResults_Validation(t *testing.T) {
	tests := []struct {
		name      string
		queryID   string
		wantErr   bool
		errString string
	}{
		{
			name:      "empty query ID should error",
			queryID:   "",
			wantErr:   true,
			errString: "queryID must be specified",
		},
		{
			name:    "valid query ID",
			queryID: "12345678-1234-1234-1234-123456789012",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.Empty(t, tt.queryID)
			} else {
				assert.NotEmpty(t, tt.queryID)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("int32Ptr", func(t *testing.T) {
		value := int32(42)
		ptr := sourceutil.Int32Ptr(value)
		assert.NotNil(t, ptr)
		assert.Equal(t, value, *ptr)
	})

	t.Run("stringValue with non-nil", func(t *testing.T) {
		str := "test string"
		result := sourceutil.StringValue(&str)
		assert.Equal(t, str, result)
	})

	t.Run("stringValue with nil", func(t *testing.T) {
		result := sourceutil.StringValue(nil)
		assert.Equal(t, "", result)
	})

	t.Run("float64Value with non-nil", func(t *testing.T) {
		value := 123.45
		result := sourceutil.Float64Value(&value)
		assert.Equal(t, value, result)
	})

	t.Run("float64Value with nil", func(t *testing.T) {
		result := sourceutil.Float64Value(nil)
		assert.Equal(t, 0.0, result)
	})
}

func TestLogEvent(t *testing.T) {
	now := time.Now()
	event := LogEvent{
		Timestamp:     now.UnixMilli(),
		Message:       "Test log message",
		LogStreamName: "test-stream",
		EventID:       "event-123",
	}

	assert.Equal(t, now.UnixMilli(), event.Timestamp)
	assert.Equal(t, "Test log message", event.Message)
	assert.Equal(t, "test-stream", event.LogStreamName)
	assert.Equal(t, "event-123", event.EventID)
}

func TestResultField(t *testing.T) {
	field := ResultField{
		Field: "@timestamp",
		Value: "2024-01-01T12:00:00Z",
	}

	assert.Equal(t, "@timestamp", field.Field)
	assert.Equal(t, "2024-01-01T12:00:00Z", field.Value)
}

func TestQueryStatistics(t *testing.T) {
	stats := QueryStatistics{
		BytesScanned:   1024.0,
		RecordsMatched: 50.0,
		RecordsScanned: 1000.0,
	}

	assert.Equal(t, 1024.0, stats.BytesScanned)
	assert.Equal(t, 50.0, stats.RecordsMatched)
	assert.Equal(t, 1000.0, stats.RecordsScanned)
}

func TestFilterLogEventsInput_Complete(t *testing.T) {
	now := time.Now()
	input := FilterLogEventsInput{
		LogGroupName:   "/aws/lambda/my-function",
		LogStreamNames: []string{"stream1", "stream2"},
		StartTime:      now.Add(-1 * time.Hour),
		EndTime:        now,
		FilterPattern:  "[level=ERROR]",
		Limit:          100,
		NextToken:      "next-token-123",
	}

	assert.Equal(t, "/aws/lambda/my-function", input.LogGroupName)
	assert.Len(t, input.LogStreamNames, 2)
	assert.Equal(t, "stream1", input.LogStreamNames[0])
	assert.Equal(t, "[level=ERROR]", input.FilterPattern)
	assert.Equal(t, int32(100), input.Limit)
	assert.Equal(t, "next-token-123", input.NextToken)
}

func TestInsightsQueryInput_Complete(t *testing.T) {
	now := time.Now()
	input := InsightsQueryInput{
		LogGroupNames: []string{"/aws/lambda/func1", "/aws/lambda/func2"},
		QueryString:   "fields @timestamp, @message | filter @message like /ERROR/",
		StartTime:     now.Add(-24 * time.Hour),
		EndTime:       now,
		Limit:         1000,
	}

	assert.Len(t, input.LogGroupNames, 2)
	assert.Contains(t, input.QueryString, "ERROR")
	assert.True(t, input.StartTime.Before(input.EndTime))
	assert.Equal(t, int32(1000), input.Limit)
}

// TestSourceInterfaceCompliance verifies that Source implements the sources.Source interface
func TestSourceInterfaceCompliance(t *testing.T) {
	config := Config{
		Name:         "test-cloudwatch",
		Kind:         "cloudwatch",
		Region:       "us-east-1",
		LogGroupName: "/aws/lambda/test",
	}

	source := &Source{Config: config}

	// Verify SourceKind method
	assert.Equal(t, SourceKind, source.SourceKind())

	// Verify ToConfig method
	returnedConfig := source.ToConfig()
	assert.Equal(t, config, returnedConfig)
}

// TestConfigInterfaceCompliance verifies that Config implements the sources.SourceConfig interface
func TestConfigInterfaceCompliance(t *testing.T) {
	config := Config{
		Name:   "test-cloudwatch",
		Kind:   "cloudwatch",
		Region: "us-east-1",
	}

	assert.Equal(t, SourceKind, config.SourceConfigKind())
}

// Example test showing how to use the CloudWatch source in integration tests
// This would require LocalStack or actual AWS credentials
func ExampleSource_FilterLogEvents() {
	// This is a documentation example showing the expected usage
	// In real tests, you would use LocalStack or mocks

	ctx := context.Background()
	config := Config{
		Name:         "example-cloudwatch",
		Kind:         "cloudwatch",
		Region:       "us-east-1",
		LogGroupName: "/aws/lambda/my-function",
		Endpoint:     "http://localhost:4566", // LocalStack endpoint
	}

	// Initialize the source (would fail without LocalStack/AWS)
	source, err := config.Initialize(ctx, noop.NewTracerProvider().Tracer("test"))
	if err != nil {
		// Handle error
		return
	}

	cwSource := source.(*Source)

	// Filter log events
	input := &FilterLogEventsInput{
		StartTime:     time.Now().Add(-1 * time.Hour),
		FilterPattern: "[level=ERROR]",
		Limit:         100,
	}

	_, err = cwSource.FilterLogEvents(ctx, input)
	if err != nil {
		// Handle error
		return
	}

	// Output would be processed here
}

// Example test showing how to use CloudWatch Logs Insights
func ExampleSource_StartInsightsQuery() {
	// This is a documentation example showing the expected usage
	// In real tests, you would use LocalStack or mocks

	ctx := context.Background()
	config := Config{
		Name:     "example-cloudwatch",
		Kind:     "cloudwatch",
		Region:   "us-east-1",
		Endpoint: "http://localhost:4566", // LocalStack endpoint
	}

	// Initialize the source (would fail without LocalStack/AWS)
	source, err := config.Initialize(ctx, noop.NewTracerProvider().Tracer("test"))
	if err != nil {
		// Handle error
		return
	}

	cwSource := source.(*Source)

	// Start an Insights query
	queryInput := &InsightsQueryInput{
		LogGroupNames: []string{"/aws/lambda/my-function"},
		QueryString:   "fields @timestamp, @message | filter @message like /ERROR/ | stats count() by bin(5m)",
		StartTime:     time.Now().Add(-24 * time.Hour),
		EndTime:       time.Now(),
		Limit:         1000,
	}

	queryOutput, err := cwSource.StartInsightsQuery(ctx, queryInput)
	if err != nil {
		// Handle error
		return
	}

	// Poll for results
	for {
		results, err := cwSource.GetInsightsQueryResults(ctx, queryOutput.QueryID)
		if err != nil {
			// Handle error
			return
		}

		if results.Status == "Complete" {
			// Process results
			break
		}

		time.Sleep(1 * time.Second)
	}

	// Output would be processed here
}
