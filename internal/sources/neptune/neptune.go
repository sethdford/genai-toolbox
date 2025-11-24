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

// Package neptune provides a source implementation for AWS Neptune graph database.
//
// This source provides Gremlin connectivity to Amazon Neptune clusters.
// It supports both standard authentication and IAM authentication with SigV4 signing.
package neptune

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/goccy/go-yaml"
	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "neptune"

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
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Endpoint string `yaml:"endpoint" validate:"required"` // wss://your-neptune-endpoint:8182/gremlin
	UseIAM   bool   `yaml:"useIAM"`                        // Enable IAM authentication
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	driver, err := initNeptuneDriver(ctx, tracer, r.Name, r.Endpoint, r.UseIAM)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create Neptune driver: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config: r,
		Driver: driver,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Driver *gremlingo.DriverRemoteConnection
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// NeptuneDriver returns the underlying Gremlin driver for direct graph operations.
func (s *Source) NeptuneDriver() *gremlingo.DriverRemoteConnection {
	return s.Driver
}

// Close closes the Neptune Gremlin connection and releases resources.
func (s *Source) Close() error {
	if s.Driver != nil {
		s.Driver.Close() // Close() doesn't return error, logs errors internally
	}
	return nil
}

// neptuneIAMAuthProvider implements gremlingo.AuthInfoProvider for Neptune IAM authentication.
// It dynamically generates SigV4-signed headers for WebSocket connections to Neptune.
type neptuneIAMAuthProvider struct {
	ctx         context.Context
	credentials aws.CredentialsProvider
	endpoint    string
	host        string
	region      string
	logger      *slog.Logger
}

// GetHeader returns HTTP headers for Neptune IAM authentication.
// It generates a SigV4-signed Authorization header for each request.
func (p *neptuneIAMAuthProvider) GetHeader() http.Header {
	// Retrieve current AWS credentials (may refresh if expired)
	creds, err := p.credentials.Retrieve(p.ctx)
	if err != nil {
		if p.logger != nil {
			p.logger.ErrorContext(p.ctx, "Failed to retrieve AWS credentials for Neptune IAM auth",
				"error", err,
				"endpoint", p.endpoint)
		}
		return http.Header{}
	}

	// Create an HTTP request for SigV4 signing
	// Neptune WebSocket connections require signing as HTTP GET requests
	httpURL := strings.Replace(p.endpoint, "wss://", "https://", 1)
	httpURL = strings.Replace(httpURL, "ws://", "http://", 1)

	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		if p.logger != nil {
			p.logger.ErrorContext(p.ctx, "Failed to create HTTP request for Neptune IAM auth",
				"error", err,
				"endpoint", httpURL)
		}
		return http.Header{}
	}

	// Set required headers for Neptune
	req.Header.Set("Host", p.host)

	// Create SigV4 signer with "neptune-db" service name
	// Neptune requires the service name to be "neptune-db" (not "neptune")
	signer := v4.NewSigner()

	// Compute payload hash (empty for GET request)
	payloadHash := sha256.Sum256([]byte(""))
	payloadHashStr := hex.EncodeToString(payloadHash[:])

	// Sign the request using AWS SigV4
	err = signer.SignHTTP(p.ctx, creds, req, payloadHashStr, "neptune-db", p.region, time.Now())
	if err != nil {
		if p.logger != nil {
			p.logger.ErrorContext(p.ctx, "Failed to sign request for Neptune IAM auth",
				"error", err,
				"region", p.region,
				"service", "neptune-db")
		}
		return http.Header{}
	}

	// Return the signed headers
	// The Authorization header contains the SigV4 signature
	return req.Header
}

// GetBasicAuth returns false as Neptune IAM authentication does not use basic auth.
func (p *neptuneIAMAuthProvider) GetBasicAuth() (ok bool, username, password string) {
	return false, "", ""
}

func initNeptuneDriver(ctx context.Context, tracer trace.Tracer, name, endpoint string, useIAM bool) (*gremlingo.DriverRemoteConnection, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// If IAM authentication is not enabled, connect without authentication
	if !useIAM {
		driver, err := gremlingo.NewDriverRemoteConnection(endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to create Neptune driver: %w", err)
		}
		return driver, nil
	}

	// IAM Authentication is enabled - implement SigV4 signing for Neptune WebSocket connections
	// Load AWS configuration using default credential chain
	// This supports: environment variables, shared config/credentials files, IAM roles, etc.
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config for IAM auth: %w", err)
	}

	// Parse the Neptune endpoint to extract host
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Neptune endpoint %q: %w", endpoint, err)
	}

	// Extract AWS region from the Neptune endpoint hostname
	// Neptune endpoints follow format: cluster-id.cluster-hash.region.neptune.amazonaws.com
	region := extractRegionFromEndpoint(parsedURL.Host)
	if region == "" {
		// Fallback to AWS config region if extraction fails
		region = cfg.Region
		if region == "" {
			return nil, fmt.Errorf("unable to determine AWS region from endpoint %q and no region in AWS config", endpoint)
		}
	}

	// Create the IAM authentication provider
	// This provider implements gremlingo.AuthInfoProvider and dynamically
	// generates SigV4-signed headers for each WebSocket connection
	authProvider := &neptuneIAMAuthProvider{
		ctx:         ctx,
		credentials: cfg.Credentials,
		endpoint:    endpoint,
		host:        parsedURL.Host,
		region:      region,
		logger:      slog.Default(),
	}

	// Create Neptune Gremlin connection with IAM authentication
	driver, err := gremlingo.NewDriverRemoteConnection(
		endpoint,
		func(settings *gremlingo.DriverRemoteConnectionSettings) {
			// Set the IAM authentication provider
			// The Gremlin driver will call GetHeader() for each connection
			settings.AuthInfo = authProvider
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create Neptune driver with IAM auth: %w", err)
	}

	return driver, nil
}

// extractRegionFromEndpoint extracts the AWS region from a Neptune endpoint hostname.
// Neptune endpoints follow the format: cluster-id.cluster-hash.region.neptune.amazonaws.com
// For example: mycluster.abc123def456.us-east-1.neptune.amazonaws.com
func extractRegionFromEndpoint(host string) string {
	parts := strings.Split(host, ".")
	// Look for the region component (typically 3rd or 4th position)
	// Region prefixes: us-, eu-, ap-, ca-, sa-, me-, af-
	for i, part := range parts {
		if strings.HasPrefix(part, "us-") || strings.HasPrefix(part, "eu-") ||
			strings.HasPrefix(part, "ap-") || strings.HasPrefix(part, "ca-") ||
			strings.HasPrefix(part, "sa-") || strings.HasPrefix(part, "me-") ||
			strings.HasPrefix(part, "af-") {
			// Validate it's actually a region by checking if followed by "neptune"
			if i < len(parts)-1 && parts[i+1] == "neptune" {
				return part
			}
		}
	}
	return ""
}
