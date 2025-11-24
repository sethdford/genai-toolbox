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

// Package tableau provides a source implementation for Tableau Server and Tableau Cloud.
//
// This source provides REST API connectivity to Tableau Server and Tableau Cloud.
// It supports both Personal Access Token (PAT) and username/password authentication.
// Authentication tokens are automatically refreshed to prevent session expiration.
package tableau

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "tableau"

// Default configuration constants
const (
	DefaultAPIVersion     = "3.27"               // Latest stable Tableau REST API version
	DefaultTimeout        = 30 * time.Second     // Default HTTP client timeout
	DefaultTokenExpiry    = 240 * time.Minute    // Tableau tokens expire after 4 hours
	TokenRefreshBuffer    = 5 * time.Minute      // Refresh token if it expires in less than 5 minutes
	MaxIdleConns          = 100                  // Maximum idle connections in pool
	MaxIdleConnsPerHost   = 10                   // Maximum idle connections per host
	IdleConnTimeout       = 90 * time.Second     // Idle connection timeout
	TLSHandshakeTimeout   = 10 * time.Second     // TLS handshake timeout
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

type Config struct {
	Name                      string `yaml:"name" validate:"required"`
	Kind                      string `yaml:"kind" validate:"required"`
	ServerURL                 string `yaml:"serverUrl" validate:"required"`          // e.g., https://tableau.example.com
	SiteName                  string `yaml:"siteName"`                               // Optional: for multi-site deployments
	Username                  string `yaml:"username"`                               // For username/password auth
	Password                  string `yaml:"password"`                               // For username/password auth
	PersonalAccessTokenName   string `yaml:"personalAccessTokenName"`                // For PAT auth
	PersonalAccessTokenSecret string `yaml:"personalAccessTokenSecret"`              // For PAT auth
	APIVersion                string `yaml:"apiVersion"`                             // Optional: defaults to latest
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initTableauClient(ctx, tracer, r.Name, r.ServerURL, r.SiteName, r.Username, r.Password, r.PersonalAccessTokenName, r.PersonalAccessTokenSecret, r.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create Tableau client: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config: r,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client *TableauClient
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// TableauClient returns the underlying Tableau REST API client for direct API access.
func (s *Source) TableauClient() *TableauClient {
	return s.Client
}

// Close signs out from Tableau and releases HTTP client resources.
func (s *Source) Close() error {
	if s == nil || s.Client == nil {
		return nil
	}
	if s.Client != nil {
		// Sign out from Tableau if we have a valid token
		if s.Client.AuthToken != "" && time.Now().Before(s.Client.TokenExpiry) {
			// Best effort sign out - don't fail if it errors
			signOutURL := fmt.Sprintf("%s/api/%s/auth/signout",
				s.Client.ServerURL, s.Client.APIVersion)
			req, err := http.NewRequest("POST", signOutURL, nil)
			if err == nil {
				req.Header.Set("X-Tableau-Auth", s.Client.AuthToken)
				resp, err := s.Client.HTTPClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}

		// Close idle HTTP connections
		if transport, ok := s.Client.HTTPClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}
	return nil
}

// TableauClient wraps HTTP client and authentication for Tableau REST API
type TableauClient struct {
	HTTPClient  *http.Client
	ServerURL   string
	SiteName    string
	APIVersion  string
	AuthToken   string
	SiteID      string
	UserID      string
	TokenExpiry time.Time

	// Store credentials for token refresh
	username                  string
	password                  string
	personalAccessTokenName   string
	personalAccessTokenSecret string
}

// Request/Response structures for authentication

// signInRequest represents the sign-in request body
type signInRequest struct {
	Credentials signInCredentials `json:"credentials" xml:"credentials"`
}

// signInCredentials holds authentication credentials
type signInCredentials struct {
	Name                      string   `json:"name,omitempty" xml:"name,attr,omitempty"`
	Password                  string   `json:"password,omitempty" xml:"password,attr,omitempty"`
	PersonalAccessTokenName   string   `json:"personalAccessTokenName,omitempty" xml:"personalAccessTokenName,attr,omitempty"`
	PersonalAccessTokenSecret string   `json:"personalAccessTokenSecret,omitempty" xml:"personalAccessTokenSecret,attr,omitempty"`
	Site                      siteInfo `json:"site" xml:"site"`
}

// siteInfo represents a Tableau site
type siteInfo struct {
	ContentUrl string `json:"contentUrl" xml:"contentUrl,attr"`
}

// signInResponse represents the sign-in response
type signInResponse struct {
	Credentials credentialsResponse `json:"credentials" xml:"credentials"`
}

// credentialsResponse holds the authentication response details
type credentialsResponse struct {
	Token                     string       `json:"token" xml:"token,attr"`
	EstimatedTimeToExpiration string       `json:"estimatedTimeToExpiration,omitempty" xml:"estimatedTimeToExpiration,attr,omitempty"`
	Site                      siteResponse `json:"site" xml:"site"`
	User                      userResponse `json:"user" xml:"user"`
}

// siteResponse represents site information in the response
type siteResponse struct {
	ID         string `json:"id" xml:"id,attr"`
	ContentUrl string `json:"contentUrl" xml:"contentUrl,attr"`
}

// userResponse represents user information in the response
type userResponse struct {
	ID string `json:"id" xml:"id,attr"`
}

// tableauError represents a Tableau REST API error
type tableauError struct {
	StatusCode int
	ErrorCode  string
	Summary    string
	Detail     string
}

func (e *tableauError) Error() string {
	return fmt.Sprintf("Tableau API error %d (code: %s): %s - %s",
		e.StatusCode, e.ErrorCode, e.Summary, e.Detail)
}

// errorResponse represents an error response from the API
type errorResponse struct {
	XMLName xml.Name `xml:"tsResponse"`
	Error   struct {
		Code    string `xml:"code,attr"`
		Summary string `xml:"summary,attr"`
		Detail  string `xml:"detail,attr"`
	} `xml:"error"`
}

func initTableauClient(ctx context.Context, tracer trace.Tracer, name, serverURL, siteName, username, password, patName, patSecret, apiVersion string) (*TableauClient, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}

	// Configure HTTP client with production-ready settings
	client := &TableauClient{
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        MaxIdleConns,
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
				IdleConnTimeout:     IdleConnTimeout,
				TLSHandshakeTimeout: TLSHandshakeTimeout,
			},
		},
		ServerURL:  serverURL,
		SiteName:   siteName,
		APIVersion: apiVersion,
	}

	// Authenticate with Tableau
	var err error
	if patName != "" && patSecret != "" {
		// Use Personal Access Token authentication (recommended)
		err = client.authenticateWithPAT(ctx, patName, patSecret)
	} else if username != "" && password != "" {
		// Use username/password authentication
		err = client.authenticateWithCredentials(ctx, username, password)
	} else {
		return nil, fmt.Errorf("authentication credentials required (username/password or PAT)")
	}

	if err != nil {
		return nil, fmt.Errorf("unable to authenticate: %w", err)
	}

	return client, nil
}

func (c *TableauClient) authenticateWithCredentials(ctx context.Context, username, password string) error {
	url := c.buildSignInURL()

	// Prepare request body
	reqBody := signInRequest{
		Credentials: signInCredentials{
			Name:     username,
			Password: password,
			Site: siteInfo{
				ContentUrl: c.SiteName,
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp.StatusCode, body)
	}

	// Store credentials for refresh
	c.username = username
	c.password = password

	// Parse and store authentication details
	return c.parseAuthResponse(body)
}

func (c *TableauClient) authenticateWithPAT(ctx context.Context, tokenName, tokenSecret string) error {
	url := c.buildSignInURL()

	// Prepare request body
	reqBody := signInRequest{
		Credentials: signInCredentials{
			PersonalAccessTokenName:   tokenName,
			PersonalAccessTokenSecret: tokenSecret,
			Site: siteInfo{
				ContentUrl: c.SiteName,
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp.StatusCode, body)
	}

	// Store credentials for refresh
	c.personalAccessTokenName = tokenName
	c.personalAccessTokenSecret = tokenSecret

	// Parse and store authentication details
	return c.parseAuthResponse(body)
}

// Helper methods

// buildSignInURL constructs the sign-in endpoint URL
func (c *TableauClient) buildSignInURL() string {
	return fmt.Sprintf("%s/api/%s/auth/signin", c.ServerURL, c.APIVersion)
}

// parseAuthResponse parses the authentication response and stores credentials
func (c *TableauClient) parseAuthResponse(body []byte) error {
	var signInResp signInResponse
	if err := json.Unmarshal(body, &signInResp); err != nil {
		return fmt.Errorf("failed to parse authentication response: %w", err)
	}

	// Store authentication details
	c.AuthToken = signInResp.Credentials.Token
	c.SiteID = signInResp.Credentials.Site.ID
	c.UserID = signInResp.Credentials.User.ID

	// Calculate token expiry
	c.TokenExpiry = time.Now().Add(DefaultTokenExpiry)

	return nil
}

// IsTokenValid checks if the authentication token is still valid.
// Returns true if the token exists and hasn't expired yet.
func (c *TableauClient) IsTokenValid() bool {
	if c.AuthToken == "" {
		return false
	}
	// Consider token invalid if it expires soon
	return time.Until(c.TokenExpiry) > TokenRefreshBuffer
}

// RefreshToken refreshes the authentication token if it's expired or about to expire.
// This method re-authenticates using the stored credentials.
func (c *TableauClient) RefreshToken(ctx context.Context) error {
	// Only refresh if token is invalid or expiring soon
	if c.IsTokenValid() {
		return nil
	}

	// Re-authenticate using stored credentials
	if c.personalAccessTokenName != "" && c.personalAccessTokenSecret != "" {
		return c.authenticateWithPAT(ctx, c.personalAccessTokenName, c.personalAccessTokenSecret)
	} else if c.username != "" && c.password != "" {
		return c.authenticateWithCredentials(ctx, c.username, c.password)
	}

	return fmt.Errorf("no credentials available for token refresh")
}

// EnsureValidToken ensures the authentication token is valid, refreshing if necessary.
// This should be called before making any API requests.
func (c *TableauClient) EnsureValidToken(ctx context.Context) error {
	if !c.IsTokenValid() {
		return c.RefreshToken(ctx)
	}
	return nil
}

// parseErrorResponse parses JSON or XML error response
func (c *TableauClient) parseErrorResponse(statusCode int, body []byte) error {
	// Try parsing as JSON first
	var jsonErr struct {
		Error struct {
			Code    string `json:"code"`
			Summary string `json:"summary"`
			Detail  string `json:"detail"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &jsonErr); err == nil && jsonErr.Error.Code != "" {
		return &tableauError{
			StatusCode: statusCode,
			ErrorCode:  jsonErr.Error.Code,
			Summary:    jsonErr.Error.Summary,
			Detail:     jsonErr.Error.Detail,
		}
	}

	// Fall back to XML parsing
	var errResp errorResponse
	if err := xml.Unmarshal(body, &errResp); err != nil {
		return &tableauError{
			StatusCode: statusCode,
			ErrorCode:  "",
			Summary:    "Failed to parse error response",
			Detail:     string(body),
		}
	}

	return &tableauError{
		StatusCode: statusCode,
		ErrorCode:  errResp.Error.Code,
		Summary:    errResp.Error.Summary,
		Detail:     errResp.Error.Detail,
	}
}
