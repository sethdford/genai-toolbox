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

// Package honeycomb provides a source implementation for Honeycomb observability.
//
// This source provides REST API connectivity to Honeycomb for querying telemetry data.
// It includes retry logic for transient failures and supports dataset management.
package honeycomb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "honeycomb"

// Default configuration constants
const (
	DefaultBaseURL      = "https://api.honeycomb.io" // Default Honeycomb API base URL
	DefaultTimeout      = 30                         // Default request timeout in seconds
	DefaultMaxRetries   = 3                          // Default number of retries for failed requests
	DefaultMaxAttempts  = 10                         // Default max attempts for polling query results
	MaxBackoffSeconds   = 10                         // Maximum backoff time for exponential backoff
)

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

// Config represents the configuration for a Honeycomb source.
type Config struct {
	Name        string `yaml:"name" validate:"required"`
	Kind        string `yaml:"kind" validate:"required"`
	APIKey      string `yaml:"apiKey" validate:"required"`      // Honeycomb API key for authentication
	Dataset     string `yaml:"dataset"`                         // Optional: default dataset
	Environment string `yaml:"environment"`                     // Optional: environment name
	BaseURL     string `yaml:"baseUrl"`                         // Optional: base URL (default: https://api.honeycomb.io)
	Timeout     int    `yaml:"timeout"`                         // Optional: request timeout in seconds (default: 30)
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initHoneycombClient(ctx, tracer, r.Name, r.APIKey, r.BaseURL, r.Timeout)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create Honeycomb client: %w", r.Name, SourceKind, err)
	}

	// Verify the connection by listing datasets
	_, err = client.ListDatasets(ctx)
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

// Source represents a Honeycomb source.
type Source struct {
	Config
	Client *Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// HoneycombClient returns the underlying Honeycomb API client for direct API access.
func (s *Source) HoneycombClient() *Client {
	return s.Client
}

// Close closes the HTTP client's idle connections and releases resources.
func (s *Source) Close() error {
	if s == nil || s.Client == nil {
		return nil
	}
	if s.Client != nil && s.Client.HTTPClient != nil {
		if transport, ok := s.Client.HTTPClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}

// Client represents a Honeycomb API client.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Dataset represents a Honeycomb dataset.
type Dataset struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Created     string `json:"created_at"`
}

// QuerySpec represents a Honeycomb query specification.
type QuerySpec struct {
	Calculations []Calculation `json:"calculations,omitempty"`
	Filters      []Filter      `json:"filters,omitempty"`
	Breakdowns   []string      `json:"breakdowns,omitempty"`
	Orders       []Order       `json:"orders,omitempty"`
	Granularity  int           `json:"granularity,omitempty"`
	TimeRange    int           `json:"time_range,omitempty"`
	StartTime    int64         `json:"start_time,omitempty"`
	EndTime      int64         `json:"end_time,omitempty"`
}

// Calculation represents a query calculation.
type Calculation struct {
	Op     string `json:"op"`
	Column string `json:"column,omitempty"`
}

// Filter represents a query filter.
type Filter struct {
	Column string      `json:"column"`
	Op     string      `json:"op"`
	Value  interface{} `json:"value"`
}

// Order represents a query result ordering.
type Order struct {
	Column string `json:"column,omitempty"`
	Op     string `json:"op,omitempty"`
	Order  string `json:"order"`
}

// Query represents a created Honeycomb query.
type Query struct {
	ID          string    `json:"id"`
	QuerySpec   QuerySpec `json:"query"`
	Created     string    `json:"created_at"`
	Updated     string    `json:"updated_at"`
}

// QueryResult represents the result of a query execution.
type QueryResult struct {
	ID        string                   `json:"id"`
	QueryID   string                   `json:"query_id"`
	Complete  bool                     `json:"complete"`
	Data      []map[string]interface{} `json:"data,omitempty"`
	Links     map[string]string        `json:"links,omitempty"`
	Error     string                   `json:"error,omitempty"`
}

func initHoneycombClient(ctx context.Context, tracer trace.Tracer, name, apiKey, baseURL string, timeout int) (*Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	// Set default base URL if not provided
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Set default timeout if not provided
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	client := &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}

	return client, nil
}

// doRequest performs an HTTP request with authentication.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("X-Honeycomb-Team", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// doRequestWithRetry wraps doRequest with retry logic for transient failures.
func (c *Client) doRequestWithRetry(ctx context.Context, method, path string, body []byte, maxRetries int) (*http.Response, error) {
	if maxRetries == 0 {
		maxRetries = DefaultMaxRetries
	}

	var lastErr error
	backoff := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		resp, err := c.doRequest(ctx, method, path, bodyReader)

		// Success or non-retryable error
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Store error for potential retry
		if err != nil {
			lastErr = err
		} else {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		}

		// Don't sleep on last attempt
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// ListDatasets lists all datasets in the Honeycomb account.
func (c *Client) ListDatasets(ctx context.Context) ([]Dataset, error) {
	resp, err := c.doRequest(ctx, "GET", "/1/datasets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var datasets []Dataset
	if err := json.NewDecoder(resp.Body).Decode(&datasets); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return datasets, nil
}

// CreateQuery creates a query in the specified dataset.
func (c *Client) CreateQuery(ctx context.Context, dataset string, spec QuerySpec) (*Query, error) {
	bodyBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query spec: %w", err)
	}

	path := fmt.Sprintf("/1/queries/%s", dataset)
	resp, err := c.doRequest(ctx, "POST", path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var query Query
	if err := json.NewDecoder(resp.Body).Decode(&query); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &query, nil
}

// ExecuteQuery executes a query and returns the result.
func (c *Client) ExecuteQuery(ctx context.Context, dataset, queryID string) (*QueryResult, error) {
	// Create query result request
	requestBody := map[string]interface{}{
		"query_id":       queryID,
		"disable_series": false,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/1/query_results/%s", dataset)
	resp, err := c.doRequest(ctx, "POST", path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetQueryResult retrieves the result of a query execution.
func (c *Client) GetQueryResult(ctx context.Context, dataset, resultID string) (*QueryResult, error) {
	path := fmt.Sprintf("/1/query_results/%s/%s", dataset, resultID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// PollQueryResult polls for query result completion with exponential backoff.
func (c *Client) PollQueryResult(ctx context.Context, dataset, resultID string, maxAttempts int) (*QueryResult, error) {
	if maxAttempts == 0 {
		maxAttempts = DefaultMaxAttempts
	}

	backoff := time.Second
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := c.GetQueryResult(ctx, dataset, resultID)
		if err != nil {
			return nil, err
		}

		if result.Complete {
			return result, nil
		}

		if result.Error != "" {
			return nil, fmt.Errorf("query failed: %s", result.Error)
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Exponential backoff with max
			backoff *= 2
			if backoff > MaxBackoffSeconds*time.Second {
				backoff = MaxBackoffSeconds * time.Second
			}
		}
	}

	return nil, fmt.Errorf("query did not complete within %d attempts", maxAttempts)
}
