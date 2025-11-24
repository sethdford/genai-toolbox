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

// Package splunk provides a source implementation for Splunk Enterprise and Splunk Cloud.
//
// This source provides REST API connectivity to Splunk for search and data ingestion.
// It supports both search API (for queries) and HTTP Event Collector (for data ingestion).
// Active search jobs are automatically tracked and cleaned up on close.
package splunk

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "splunk"

// Default configuration constants
const (
	DefaultPort         = 8089   // Default Splunk management port
	DefaultHECPort      = 8088   // Default HTTP Event Collector port
	DefaultScheme       = "https" // Default connection scheme
	DefaultTimeout      = "120s"  // Default client timeout
)

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{
		Name:    name,
		Timeout: DefaultTimeout,
		Port:    DefaultPort,
		HECPort: DefaultHECPort,
		Scheme:  DefaultScheme,
	}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

// Config represents the configuration for a Splunk source.
// It supports both token-based and username/password authentication.
type Config struct {
	Name                   string `yaml:"name" validate:"required"`
	Kind                   string `yaml:"kind" validate:"required"`
	Host                   string `yaml:"host" validate:"required"`
	Port                   int    `yaml:"port"`
	HECPort                int    `yaml:"hecPort"`
	Scheme                 string `yaml:"scheme"`
	Token                  string `yaml:"token"`
	Username               string `yaml:"username"`
	Password               string `yaml:"password"`
	HECToken               string `yaml:"hecToken"`
	Timeout                string `yaml:"timeout"`
	DisableSslVerification bool   `yaml:"disableSslVerification"`
}

func (c Config) SourceConfigKind() string {
	return SourceKind
}

// Source represents an initialized Splunk source with an HTTP client.
type Source struct {
	Config
	Client     *http.Client
	baseURL    string
	hecURL     string
	authToken  string
	activeJobs sync.Map // Track active search job SIDs
}

var _ sources.Source = &Source{}

// Initialize creates a new Splunk Source instance.
func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to get logger from context: %w", c.Name, SourceKind, err)
	}

	// Parse timeout
	duration, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to parse timeout string as time.Duration: %w", c.Name, SourceKind, err)
	}

	// Configure HTTP transport
	tr := &http.Transport{}
	if c.DisableSslVerification {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		logger.WarnContext(ctx, "Insecure HTTP is enabled for Splunk source %s. TLS certificate verification is skipped.", c.Name)
	}

	client := &http.Client{
		Timeout:   duration,
		Transport: tr,
	}

	// Build base URLs
	baseURL := fmt.Sprintf("%s://%s:%d", c.Scheme, c.Host, c.Port)
	hecURL := fmt.Sprintf("%s://%s:%d", c.Scheme, c.Host, c.HECPort)

	s := &Source{
		Config:  c,
		Client:  client,
		baseURL: baseURL,
		hecURL:  hecURL,
	}

	// Authenticate and get session key if using username/password
	if c.Token != "" {
		// Use token-based authentication
		s.authToken = c.Token
		logger.DebugContext(ctx, "Using token-based authentication for Splunk source %s", c.Name)
	} else if c.Username != "" && c.Password != "" {
		// Use username/password authentication to get session key
		sessionKey, err := s.authenticate(ctx)
		if err != nil {
			return nil, fmt.Errorf("source %q (%s): authentication failed: %w", c.Name, SourceKind, err)
		}
		s.authToken = sessionKey
		logger.DebugContext(ctx, "Successfully authenticated with username/password for Splunk source %s", c.Name)
	} else {
		return nil, fmt.Errorf("source %q (%s): requires either token or username/password authentication", c.Name, SourceKind)
	}

	// Test connection
	if err := s.testConnection(ctx); err != nil {
		return nil, fmt.Errorf("source %q (%s): connection test failed: %w", c.Name, SourceKind, err)
	}

	logger.DebugContext(ctx, "Successfully connected to Splunk source %s", c.Name)
	return s, nil
}

// authenticate obtains a session key using username/password authentication.
func (s *Source) authenticate(ctx context.Context) (string, error) {
	authURL := fmt.Sprintf("%s/services/auth/login", s.baseURL)

	data := url.Values{}
	data.Set("username", s.Username)
	data.Set("password", s.Password)
	data.Set("output_mode", "json")

	req, err := http.NewRequestWithContext(ctx, "POST", authURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create authentication request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp struct {
		SessionKey string `json:"sessionKey"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode authentication response: %w", err)
	}

	if authResp.SessionKey == "" {
		return "", fmt.Errorf("no session key returned from authentication")
	}

	return authResp.SessionKey, nil
}

// testConnection verifies the connection to Splunk by making a simple API call.
func (s *Source) testConnection(ctx context.Context) error {
	testURL := fmt.Sprintf("%s/services/server/info?output_mode=json", s.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.authToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("test request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection test failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SourceKind returns the kind string for this source.
func (s *Source) SourceKind() string {
	return SourceKind
}

// ToConfig returns the configuration for this source.
func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// SplunkClient returns the underlying HTTP client for direct API access.
func (s *Source) SplunkClient() *http.Client {
	return s.Client
}

// BaseURL returns the base URL for Splunk REST API endpoints.
func (s *Source) BaseURL() string {
	return s.baseURL
}

// HECURL returns the base URL for HTTP Event Collector endpoints.
func (s *Source) HECURL() string {
	return s.hecURL
}

// AuthToken returns the authentication token for API requests.
func (s *Source) AuthToken() string {
	return s.authToken
}

// Close releases resources and closes HTTP client connections.
func (s *Source) Close() error {
	if s == nil || s.Client == nil {
		return nil
	}
	// Cancel all active search jobs
	s.activeJobs.Range(func(key, value interface{}) bool {
		if sid, ok := key.(string); ok {
			_ = s.DeleteSearchJob(context.Background(), sid)
		}
		return true
	})

	if s.Client != nil {
		if transport, ok := s.Client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}

// SearchJob represents a Splunk search job.
type SearchJob struct {
	SID string `json:"sid"`
}

// SearchJobResponse represents the response from creating a search job.
type SearchJobResponse struct {
	SID string `json:"sid"`
}

// CreateSearchJob creates a new search job in Splunk.
// The search parameter should be a valid SPL (Search Processing Language) query.
// Example: "search index=main error | head 100"
func (s *Source) CreateSearchJob(ctx context.Context, search string, params map[string]string) (*SearchJobResponse, error) {
	searchURL := fmt.Sprintf("%s/services/search/jobs", s.baseURL)

	data := url.Values{}
	data.Set("search", search)
	data.Set("output_mode", "json")

	// Add any additional parameters
	for k, v := range params {
		data.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create search job request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.authToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search job request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create search job with status %d: %s", resp.StatusCode, string(body))
	}

	var jobResp SearchJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return nil, fmt.Errorf("failed to decode search job response: %w", err)
	}

	if jobResp.SID != "" {
		s.activeJobs.Store(jobResp.SID, true)
	}

	return &jobResp, nil
}

// SearchJobStatus represents the status of a search job.
type SearchJobStatus struct {
	Entry []struct {
		Content struct {
			IsDone      bool    `json:"isDone"`
			IsFinalized bool    `json:"isFinalized"`
			IsFailed    bool    `json:"isFailed"`
			Progress    float64 `json:"doneProgress"`
			ResultCount int     `json:"resultCount"`
		} `json:"content"`
	} `json:"entry"`
}

// GetSearchJobStatus retrieves the status of a search job.
func (s *Source) GetSearchJobStatus(ctx context.Context, sid string) (*SearchJobStatus, error) {
	statusURL := fmt.Sprintf("%s/services/search/jobs/%s?output_mode=json", s.baseURL, sid)

	req, err := http.NewRequestWithContext(ctx, "GET", statusURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.authToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get job status with status %d: %s", resp.StatusCode, string(body))
	}

	var status SearchJobStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// GetSearchResults retrieves the results of a completed search job.
func (s *Source) GetSearchResults(ctx context.Context, sid string, offset int, count int) ([]byte, error) {
	resultsURL := fmt.Sprintf("%s/services/search/jobs/%s/results?output_mode=json&offset=%d&count=%d",
		s.baseURL, sid, offset, count)

	req, err := http.NewRequestWithContext(ctx, "GET", resultsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create results request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.authToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("results request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get results with status %d: %s", resp.StatusCode, string(body))
	}

	results, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read results: %w", err)
	}

	return results, nil
}

// DeleteSearchJob deletes a search job.
func (s *Source) DeleteSearchJob(ctx context.Context, sid string) error {
	deleteURL := fmt.Sprintf("%s/services/search/jobs/%s", s.baseURL, sid)

	req, err := http.NewRequestWithContext(ctx, "DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.authToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete job with status %d: %s", resp.StatusCode, string(body))
	}

	s.activeJobs.Delete(sid)

	return nil
}

// HECEvent represents a single event for HTTP Event Collector.
type HECEvent struct {
	Time       *int64                 `json:"time,omitempty"`
	Host       string                 `json:"host,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourceType string                 `json:"sourcetype,omitempty"`
	Index      string                 `json:"index,omitempty"`
	Event      interface{}            `json:"event"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
}

// SendHECEvent sends an event to the HTTP Event Collector.
// Requires HECToken to be configured.
func (s *Source) SendHECEvent(ctx context.Context, event *HECEvent) error {
	if s.HECToken == "" {
		return fmt.Errorf("HEC token not configured")
	}

	hecURL := fmt.Sprintf("%s/services/collector/event", s.hecURL)

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hecURL, strings.NewReader(string(eventJSON)))
	if err != nil {
		return fmt.Errorf("failed to create HEC request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.HECToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("HEC request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HEC request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendHECRawEvent sends a raw event to the HTTP Event Collector.
// Requires HECToken to be configured.
func (s *Source) SendHECRawEvent(ctx context.Context, event string, params map[string]string) error {
	if s.HECToken == "" {
		return fmt.Errorf("HEC token not configured")
	}

	hecURL := fmt.Sprintf("%s/services/collector/raw", s.hecURL)

	// Add query parameters if provided
	if len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Set(k, v)
		}
		hecURL = fmt.Sprintf("%s?%s", hecURL, values.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, "POST", hecURL, strings.NewReader(event))
	if err != nil {
		return fmt.Errorf("failed to create HEC raw request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", s.HECToken))

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("HEC raw request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HEC raw request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
