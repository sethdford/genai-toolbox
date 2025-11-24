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

// Package timestream provides a source implementation for AWS Timestream.
//
// This source provides access to Amazon Timestream time-series database.
// It includes both query and write clients for full time-series data operations.
package timestream

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/timestreamquery"
	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "timestream"

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
	Name            string `yaml:"name" validate:"required"`
	Kind            string `yaml:"kind" validate:"required"`
	Region          string `yaml:"region" validate:"required"`
	Database        string `yaml:"database"`        // Optional: default database name
	AccessKeyID     string `yaml:"accessKeyId"`     // Optional: explicit credentials
	SecretAccessKey string `yaml:"secretAccessKey"` // Optional: explicit credentials
	SessionToken    string `yaml:"sessionToken"`    // Optional: session token
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	queryClient, writeClient, err := initTimestreamClients(ctx, tracer, r.Name, r.Region, r.AccessKeyID, r.SecretAccessKey, r.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create Timestream clients: %w", r.Name, SourceKind, err)
	}

	// Verify the connection by listing databases
	_, err = writeClient.ListDatabases(ctx, &timestreamwrite.ListDatabasesInput{})
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to connect successfully: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config:      r,
		QueryClient: queryClient,
		WriteClient: writeClient,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	QueryClient *timestreamquery.Client
	WriteClient *timestreamwrite.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// TimestreamQueryClient returns the underlying AWS Timestream Query client for direct API access.
func (s *Source) TimestreamQueryClient() *timestreamquery.Client {
	return s.QueryClient
}

// TimestreamWriteClient returns the underlying AWS Timestream Write client for direct API access.
func (s *Source) TimestreamWriteClient() *timestreamwrite.Client {
	return s.WriteClient
}

// Close is not needed for this source because AWS SDK v2 clients manage
// their own connection pooling and cleanup automatically.

func initTimestreamClients(ctx context.Context, tracer trace.Tracer, name, region, accessKeyID, secretAccessKey, sessionToken string) (*timestreamquery.Client, *timestreamwrite.Client, error) {
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
		return nil, nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create Timestream clients
	queryClient := timestreamquery.NewFromConfig(cfg)
	writeClient := timestreamwrite.NewFromConfig(cfg)

	return queryClient, writeClient, nil
}
