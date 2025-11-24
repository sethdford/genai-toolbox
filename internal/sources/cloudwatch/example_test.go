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

package cloudwatch_test

import (
	"context"
	"fmt"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources/cloudwatch"
	"go.opentelemetry.io/otel/trace/noop"
)

// ExampleConfig_basic demonstrates basic CloudWatch Logs configuration
func ExampleConfig_basic() {
	config := cloudwatch.Config{
		Name:   "my-cloudwatch",
		Kind:   "cloudwatch",
		Region: "us-east-1",
	}

	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("example")

	// Initialize would normally connect to AWS
	// source, err := config.Initialize(ctx, tracer)
	_, _ = config.Initialize(ctx, tracer)

	fmt.Println("CloudWatch source configured for region:", config.Region)
	// Output:
	// CloudWatch source configured for region: us-east-1
}

// ExampleConfig_withLogGroup demonstrates configuration with a default log group
func ExampleConfig_withLogGroup() {
	config := cloudwatch.Config{
		Name:         "lambda-logs",
		Kind:         "cloudwatch",
		Region:       "us-west-2",
		LogGroupName: "/aws/lambda/my-function",
	}

	fmt.Println("Default log group:", config.LogGroupName)
	// Output:
	// Default log group: /aws/lambda/my-function
}

// ExampleSource_FilterLogEvents demonstrates filtering log events
func ExampleSource_FilterLogEvents() {
	// This example shows the API usage pattern
	// In actual use, you would need valid AWS credentials

	ctx := context.Background()

	// Normally you would initialize from config:
	// config := cloudwatch.Config{...}
	// source, err := config.Initialize(ctx, tracer)

	// For demonstration, we'll show the input structure
	input := &cloudwatch.FilterLogEventsInput{
		LogGroupName:  "/aws/lambda/my-function",
		StartTime:     time.Now().Add(-1 * time.Hour),
		EndTime:       time.Now(),
		FilterPattern: "[level=ERROR]",
		Limit:         100,
	}

	fmt.Printf("Filtering logs from: %s\n", input.LogGroupName)
	fmt.Printf("Time range: last %v\n", time.Hour)
	fmt.Printf("Filter pattern: %s\n", input.FilterPattern)

	// In actual use:
	// output, err := source.FilterLogEvents(ctx, input)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// for _, event := range output.Events {
	//     fmt.Println(event.Message)
	// }

	_ = ctx // use the context
	// Output:
	// Filtering logs from: /aws/lambda/my-function
	// Time range: last 1h0m0s
	// Filter pattern: [level=ERROR]
}

// ExampleSource_StartInsightsQuery demonstrates CloudWatch Logs Insights queries
func ExampleSource_StartInsightsQuery() {
	// This example shows the API usage pattern
	// In actual use, you would need valid AWS credentials

	ctx := context.Background()

	// Create an Insights query
	input := &cloudwatch.InsightsQueryInput{
		LogGroupNames: []string{"/aws/lambda/my-function"},
		QueryString: `
			fields @timestamp, @message
			| filter @message like /ERROR/
			| stats count() as error_count by bin(5m)
			| sort @timestamp desc
			| limit 100
		`,
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Limit:     1000,
	}

	fmt.Println("Query target:", input.LogGroupNames[0])
	fmt.Println("Looking for: ERROR messages")
	fmt.Println("Aggregation: 5-minute bins")

	// In actual use:
	// queryOutput, err := source.StartInsightsQuery(ctx, input)
	// if err != nil {
	//     log.Fatal(err)
	// }
	//
	// // Poll for results
	// for {
	//     results, err := source.GetInsightsQueryResults(ctx, queryOutput.QueryID)
	//     if err != nil {
	//         log.Fatal(err)
	//     }
	//     if results.Status == "Complete" {
	//         // Process results
	//         break
	//     }
	//     time.Sleep(1 * time.Second)
	// }

	_ = ctx // use the context
	// Output:
	// Query target: /aws/lambda/my-function
	// Looking for: ERROR messages
	// Aggregation: 5-minute bins
}

// ExampleSource_ListLogGroups demonstrates discovering log groups
func ExampleSource_ListLogGroups() {
	// This example shows the API usage pattern

	fmt.Println("Listing CloudWatch log groups...")
	fmt.Println("Usage pattern:")
	fmt.Println("  logGroups, nextToken, err := source.ListLogGroups(ctx, 50, \"\")")
	fmt.Println("  for _, lg := range logGroups {")
	fmt.Println("      fmt.Println(lg)")
	fmt.Println("  }")

	// Output:
	// Listing CloudWatch log groups...
	// Usage pattern:
	//   logGroups, nextToken, err := source.ListLogGroups(ctx, 50, "")
	//   for _, lg := range logGroups {
	//       fmt.Println(lg)
	//   }
}

// ExampleFilterLogEventsInput demonstrates creating filter input
func ExampleFilterLogEventsInput() {
	input := &cloudwatch.FilterLogEventsInput{
		LogGroupName:   "/aws/lambda/my-function",
		LogStreamNames: []string{"2024/01/01/[$LATEST]abc123"},
		StartTime:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC),
		FilterPattern:  "[level=ERROR]",
		Limit:          100,
	}

	fmt.Printf("Log group: %s\n", input.LogGroupName)
	fmt.Printf("Streams: %d\n", len(input.LogStreamNames))
	fmt.Printf("Filter: %s\n", input.FilterPattern)
	fmt.Printf("Limit: %d\n", input.Limit)

	// Output:
	// Log group: /aws/lambda/my-function
	// Streams: 1
	// Filter: [level=ERROR]
	// Limit: 100
}

// ExampleInsightsQueryInput demonstrates creating Insights query input
func ExampleInsightsQueryInput() {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	input := &cloudwatch.InsightsQueryInput{
		LogGroupNames: []string{
			"/aws/lambda/function-1",
			"/aws/lambda/function-2",
		},
		QueryString: "fields @timestamp, @message | filter @message like /ERROR/ | stats count() by bin(1h)",
		StartTime:   now.Add(-24 * time.Hour),
		EndTime:     now,
		Limit:       1000,
	}

	fmt.Printf("Querying %d log groups\n", len(input.LogGroupNames))
	fmt.Printf("Max results: %d\n", input.Limit)
	fmt.Printf("Time span: %v\n", input.EndTime.Sub(input.StartTime))

	// Output:
	// Querying 2 log groups
	// Max results: 1000
	// Time span: 24h0m0s
}

// ExampleLogEvent demonstrates working with log events
func ExampleLogEvent() {
	event := cloudwatch.LogEvent{
		Timestamp:     time.Now().UnixMilli(),
		Message:       "ERROR: Connection timeout",
		LogStreamName: "2024/01/01/[$LATEST]abc123",
		EventID:       "event-12345",
	}

	eventTime := time.UnixMilli(event.Timestamp)
	fmt.Printf("Time: %s\n", eventTime.Format(time.RFC3339))
	fmt.Printf("Stream: %s\n", event.LogStreamName)
	fmt.Printf("Message: %s\n", event.Message)

	// Output will vary due to timestamp, but format would be:
	// Time: 2024-01-01T12:00:00Z
	// Stream: 2024/01/01/[$LATEST]abc123
	// Message: ERROR: Connection timeout
}

// ExampleInsightsResultsOutput demonstrates processing query results
func ExampleInsightsResultsOutput() {
	// Simulated query results structure
	results := &cloudwatch.InsightsResultsOutput{
		Status: "Complete",
		Results: [][]cloudwatch.ResultField{
			{
				{Field: "@timestamp", Value: "2024-01-01T12:00:00Z"},
				{Field: "error_count", Value: "42"},
			},
			{
				{Field: "@timestamp", Value: "2024-01-01T13:00:00Z"},
				{Field: "error_count", Value: "37"},
			},
		},
		Statistics: &cloudwatch.QueryStatistics{
			BytesScanned:   1024000,
			RecordsMatched: 79,
			RecordsScanned: 150000,
		},
	}

	fmt.Printf("Query status: %s\n", results.Status)
	fmt.Printf("Result rows: %d\n", len(results.Results))
	fmt.Printf("Bytes scanned: %.0f\n", results.Statistics.BytesScanned)
	fmt.Printf("Records matched: %.0f\n", results.Statistics.RecordsMatched)

	// Output:
	// Query status: Complete
	// Result rows: 2
	// Bytes scanned: 1024000
	// Records matched: 79
}
