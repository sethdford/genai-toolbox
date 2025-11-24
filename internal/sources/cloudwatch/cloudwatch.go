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

// Package cloudwatch provides a source implementation for AWS CloudWatch Logs.
//
// This source provides connectivity to Amazon CloudWatch Logs for log querying and analysis.
// It supports both FilterLogEvents and CloudWatch Logs Insights queries.
package cloudwatch

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	sourceutil "github.com/googleapis/genai-toolbox/internal/sources/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cloudwatch"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

// Config represents the configuration for a CloudWatch Logs source.
// It provides access to AWS CloudWatch Logs for querying and streaming log data.
type Config struct {
	Name            string `yaml:"name" validate:"required"`
	Kind            string `yaml:"kind" validate:"required"`
	Region          string `yaml:"region" validate:"required"`
	LogGroupName    string `yaml:"logGroupName"` // Optional: default log group to query
	Endpoint        string `yaml:"endpoint"`     // Optional: for custom endpoints (e.g., LocalStack)
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	SessionToken    string `yaml:"sessionToken"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize creates a new CloudWatch Logs source from the configuration.
// It establishes a connection to AWS CloudWatch Logs and verifies connectivity
// by attempting to describe log groups.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initCloudWatchLogsClient(ctx, tracer, r.Name, r.Region, r.Endpoint, r.AccessKeyID, r.SecretAccessKey, r.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create CloudWatch Logs client: %w", err)
	}

	// Verify the connection by describing log groups
	_, err = client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		Limit: sourceutil.Int32Ptr(1),
	})
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to connect successfully: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config: r,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

// Source represents an active CloudWatch Logs source connection.
// It provides methods for querying and streaming CloudWatch Logs data.
type Source struct {
	Config
	Client *cloudwatchlogs.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// CloudWatchLogsClient returns the underlying CloudWatch Logs client.
// This allows direct access to the AWS SDK client for advanced operations.
func (s *Source) CloudWatchLogsClient() *cloudwatchlogs.Client {
	return s.Client
}

// FilterLogEventsInput represents the input parameters for filtering log events.
type FilterLogEventsInput struct {
	LogGroupName   string    // Required: The name of the log group to query
	LogStreamNames []string  // Optional: Specific log streams to query
	StartTime      time.Time // Optional: Start of time range
	EndTime        time.Time // Optional: End of time range
	FilterPattern  string    // Optional: CloudWatch Logs filter pattern
	Limit          int32     // Optional: Maximum number of events to return
	NextToken      string    // Optional: Token for pagination
}

// FilterLogEventsOutput represents the output from filtering log events.
type FilterLogEventsOutput struct {
	Events        []LogEvent
	NextToken     *string
	SearchedBytes *int64 // Always nil - FilterLogEvents API doesn't provide this metric
}

// LogEvent represents a single CloudWatch log event.
type LogEvent struct {
	Timestamp     int64  // The time the event occurred (milliseconds since epoch)
	Message       string // The log message
	LogStreamName string // The log stream that contains this event
	EventID       string // The unique identifier for this event
}

// FilterLogEvents retrieves log events from CloudWatch Logs using the FilterLogEvents API.
// This is suitable for simple queries and real-time log streaming.
//
// Example usage:
//
//	input := &FilterLogEventsInput{
//	    LogGroupName: "/aws/lambda/my-function",
//	    StartTime: time.Now().Add(-1 * time.Hour),
//	    FilterPattern: "[level=ERROR]",
//	    Limit: 100,
//	}
//	output, err := source.FilterLogEvents(ctx, input)
func (s *Source) FilterLogEvents(ctx context.Context, input *FilterLogEventsInput) (*FilterLogEventsOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	logGroupName := input.LogGroupName
	if logGroupName == "" {
		logGroupName = s.LogGroupName
	}
	if logGroupName == "" {
		return nil, fmt.Errorf("logGroupName must be specified")
	}

	filterInput := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
	}

	if len(input.LogStreamNames) > 0 {
		filterInput.LogStreamNames = input.LogStreamNames
	}

	if !input.StartTime.IsZero() {
		startTimeMs := input.StartTime.UnixMilli()
		filterInput.StartTime = &startTimeMs
	}

	if !input.EndTime.IsZero() {
		endTimeMs := input.EndTime.UnixMilli()
		filterInput.EndTime = &endTimeMs
	}

	if input.FilterPattern != "" {
		filterInput.FilterPattern = &input.FilterPattern
	}

	if input.Limit > 0 {
		filterInput.Limit = &input.Limit
	}

	if input.NextToken != "" {
		filterInput.NextToken = &input.NextToken
	}

	output, err := s.Client.FilterLogEvents(ctx, filterInput)
	if err != nil {
		return nil, fmt.Errorf("failed to filter log events: %w", err)
	}

	events := make([]LogEvent, 0, len(output.Events))
	for _, event := range output.Events {
		logEvent := LogEvent{
			EventID: sourceutil.StringValue(event.EventId),
		}
		if event.Timestamp != nil {
			logEvent.Timestamp = *event.Timestamp
		}
		if event.Message != nil {
			logEvent.Message = *event.Message
		}
		if event.LogStreamName != nil {
			logEvent.LogStreamName = *event.LogStreamName
		}
		events = append(events, logEvent)
	}

	return &FilterLogEventsOutput{
		Events:        events,
		NextToken:     output.NextToken,
		SearchedBytes: nil, // Not available from FilterLogEvents API
	}, nil
}

// InsightsQueryInput represents the input parameters for running a CloudWatch Logs Insights query.
type InsightsQueryInput struct {
	LogGroupNames []string  // Required: Log groups to query
	QueryString   string    // Required: CloudWatch Logs Insights query
	StartTime     time.Time // Required: Start of time range
	EndTime       time.Time // Required: End of time range
	Limit         int32     // Optional: Maximum number of log events to return
}

// InsightsQueryOutput represents the output from a CloudWatch Logs Insights query.
type InsightsQueryOutput struct {
	QueryID string // The unique identifier for the query
}

// InsightsResultsOutput represents the results from a CloudWatch Logs Insights query.
type InsightsResultsOutput struct {
	Status     string           // Query status: Running, Complete, Failed, Cancelled
	Results    [][]ResultField  // Query results as rows of fields
	Statistics *QueryStatistics // Query execution statistics
}

// ResultField represents a single field in a query result row.
type ResultField struct {
	Field string // The field name
	Value string // The field value
}

// QueryStatistics contains statistics about query execution.
type QueryStatistics struct {
	BytesScanned   float64 // Number of bytes scanned
	RecordsMatched float64 // Number of log events matched
	RecordsScanned float64 // Number of log events scanned
}

// StartInsightsQuery starts a CloudWatch Logs Insights query.
// This is suitable for complex analytical queries over historical log data.
// After starting a query, use GetInsightsQueryResults to retrieve the results.
//
// Example usage:
//
//	input := &InsightsQueryInput{
//	    LogGroupNames: []string{"/aws/lambda/my-function"},
//	    QueryString: "fields @timestamp, @message | filter @message like /ERROR/ | stats count() by bin(5m)",
//	    StartTime: time.Now().Add(-24 * time.Hour),
//	    EndTime: time.Now(),
//	    Limit: 1000,
//	}
//	output, err := source.StartInsightsQuery(ctx, input)
func (s *Source) StartInsightsQuery(ctx context.Context, input *InsightsQueryInput) (*InsightsQueryOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	if len(input.LogGroupNames) == 0 {
		if s.LogGroupName != "" {
			input.LogGroupNames = []string{s.LogGroupName}
		} else {
			return nil, fmt.Errorf("logGroupNames must be specified")
		}
	}

	if input.QueryString == "" {
		return nil, fmt.Errorf("queryString must be specified")
	}

	if input.StartTime.IsZero() || input.EndTime.IsZero() {
		return nil, fmt.Errorf("startTime and endTime must be specified")
	}

	startTimeUnix := input.StartTime.Unix()
	endTimeUnix := input.EndTime.Unix()

	queryInput := &cloudwatchlogs.StartQueryInput{
		LogGroupNames: input.LogGroupNames,
		QueryString:   &input.QueryString,
		StartTime:     &startTimeUnix,
		EndTime:       &endTimeUnix,
	}

	if input.Limit > 0 {
		limit := int32(input.Limit)
		queryInput.Limit = &limit
	}

	output, err := s.Client.StartQuery(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to start insights query: %w", err)
	}

	return &InsightsQueryOutput{
		QueryID: sourceutil.StringValue(output.QueryId),
	}, nil
}

// GetInsightsQueryResults retrieves the results of a CloudWatch Logs Insights query.
// The query must have been started using StartInsightsQuery.
// You may need to poll this method until the query status is "Complete".
//
// Example usage:
//
//	results, err := source.GetInsightsQueryResults(ctx, queryID)
//	if err != nil {
//	    return err
//	}
//	if results.Status == "Complete" {
//	    for _, row := range results.Results {
//	        for _, field := range row {
//	            fmt.Printf("%s: %s\n", field.Field, field.Value)
//	        }
//	    }
//	}
func (s *Source) GetInsightsQueryResults(ctx context.Context, queryID string) (*InsightsResultsOutput, error) {
	if queryID == "" {
		return nil, fmt.Errorf("queryID must be specified")
	}

	output, err := s.Client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
		QueryId: &queryID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get query results: %w", err)
	}

	results := make([][]ResultField, 0, len(output.Results))
	for _, row := range output.Results {
		fields := make([]ResultField, 0, len(row))
		for _, field := range row {
			fields = append(fields, ResultField{
				Field: sourceutil.StringValue(field.Field),
				Value: sourceutil.StringValue(field.Value),
			})
		}
		results = append(results, fields)
	}

	var statistics *QueryStatistics
	if output.Statistics != nil {
		statistics = &QueryStatistics{
			BytesScanned:   output.Statistics.BytesScanned,
			RecordsMatched: output.Statistics.RecordsMatched,
			RecordsScanned: output.Statistics.RecordsScanned,
		}
	}

	return &InsightsResultsOutput{
		Status:     string(output.Status),
		Results:    results,
		Statistics: statistics,
	}, nil
}

// ListLogGroups returns a list of log groups in the account.
// This is useful for discovering available log groups to query.
func (s *Source) ListLogGroups(ctx context.Context, limit int32, nextToken string) ([]string, string, error) {
	input := &cloudwatchlogs.DescribeLogGroupsInput{}

	if limit > 0 {
		input.Limit = &limit
	}

	if nextToken != "" {
		input.NextToken = &nextToken
	}

	output, err := s.Client.DescribeLogGroups(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list log groups: %w", err)
	}

	logGroups := make([]string, 0, len(output.LogGroups))
	for _, lg := range output.LogGroups {
		if lg.LogGroupName != nil {
			logGroups = append(logGroups, *lg.LogGroupName)
		}
	}

	return logGroups, sourceutil.StringValue(output.NextToken), nil
}

// ListLogStreams returns a list of log streams in a log group.
// This is useful for discovering available log streams to query.
func (s *Source) ListLogStreams(ctx context.Context, logGroupName string, limit int32, nextToken string) ([]types.LogStream, string, error) {
	if logGroupName == "" {
		logGroupName = s.LogGroupName
	}
	if logGroupName == "" {
		return nil, "", fmt.Errorf("logGroupName must be specified")
	}

	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroupName,
	}

	if limit > 0 {
		input.Limit = &limit
	}

	if nextToken != "" {
		input.NextToken = &nextToken
	}

	output, err := s.Client.DescribeLogStreams(ctx, input)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list log streams: %w", err)
	}

	return output.LogStreams, sourceutil.StringValue(output.NextToken), nil
}

// initCloudWatchLogsClient initializes an AWS CloudWatch Logs client with the provided configuration.
// It supports both default AWS credential chain and explicit credentials.
func initCloudWatchLogsClient(ctx context.Context, tracer trace.Tracer, name, region, endpoint, accessKeyID, secretAccessKey, sessionToken string) (*cloudwatchlogs.Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build AWS config load options
	configOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// Use explicit credentials if provided
	if accessKeyID != "" && secretAccessKey != "" {
		configOpts = append(configOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, sessionToken),
		))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create CloudWatch Logs client options
	opts := []func(*cloudwatchlogs.Options){}

	// Add custom endpoint if specified (for LocalStack or custom endpoints)
	if endpoint != "" {
		opts = append(opts, func(o *cloudwatchlogs.Options) {
			o.BaseEndpoint = &endpoint
		})
	}

	// Create the CloudWatch Logs client
	client := cloudwatchlogs.NewFromConfig(cfg, opts...)

	return client, nil
}
