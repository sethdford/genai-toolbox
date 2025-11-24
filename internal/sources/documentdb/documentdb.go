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

// Package documentdb provides a source implementation for AWS DocumentDB.
//
// This source provides MongoDB-compatible connectivity to Amazon DocumentDB clusters.
// TLS connections are supported via CA certificate configuration.
package documentdb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "documentdb"

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
	Name      string `yaml:"name" validate:"required"`
	Kind      string `yaml:"kind" validate:"required"`
	Uri       string `yaml:"uri" validate:"required"` // DocumentDB connection URI
	TLSCAFile string `yaml:"tlsCAFile"`               // Path to CA certificate for TLS
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initDocumentDBClient(ctx, tracer, r.Name, r.Uri, r.TLSCAFile)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create DocumentDB client: %w", r.Name, SourceKind, err)
	}

	// Verify the connection
	err = client.Ping(ctx, nil)
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

type Source struct {
	Config
	Client *mongo.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// DocumentDBClient returns the underlying MongoDB client for direct API access.
func (s *Source) DocumentDBClient() *mongo.Client {
	return s.Client
}

// Close disconnects from DocumentDB and releases resources.
func (s *Source) Close() error {
	if s.Client != nil {
		return s.Client.Disconnect(context.Background())
	}
	return nil
}

func initDocumentDBClient(ctx context.Context, tracer trace.Tracer, name, uri, tlsCAFile string) (*mongo.Client, error) {
	// Start a tracing span
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		userAgent = "genai-toolbox"
	}

	// Create client options
	clientOpts := options.Client().ApplyURI(uri).SetAppName(userAgent)

	// DocumentDB requires TLS
	if tlsCAFile != "" {
		// Set TLS config with CA file
		tlsConfig, err := loadTLSConfig(tlsCAFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load TLS config: %w", err)
		}
		clientOpts.SetTLSConfig(tlsConfig)
	}

	// Create a new MongoDB client (DocumentDB is MongoDB-compatible)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create DocumentDB client: %w", err)
	}

	return client, nil
}

// loadTLSConfig loads TLS configuration from a CA certificate file.
// Uses os.ReadFile instead of deprecated ioutil.ReadFile (Go 1.16+).
func loadTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	if caFile != "" {
		certs := x509.NewCertPool()

		// Use os.ReadFile instead of deprecated ioutil.ReadFile
		pemData, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read CA file: %w", err)
		}

		if !certs.AppendCertsFromPEM(pemData) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}

		tlsConfig.RootCAs = certs
	}

	return tlsConfig, nil
}
