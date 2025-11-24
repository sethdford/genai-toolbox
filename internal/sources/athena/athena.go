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

// Package athena provides a source implementation for AWS Athena serverless SQL.
//
// This source provides connectivity to Amazon Athena for serverless SQL queries.
// Query results are stored in S3 and encryption options are configurable.
package athena

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	sourceutil "github.com/googleapis/genai-toolbox/internal/sources/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "athena"

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

// Config holds the Athena source configuration.
// Note: Fields like Database, OutputLocation, WorkGroup, etc. are stored for use by
// consuming code when executing queries. They are not used during client initialization,
// which only requires Region for authentication and connection setup.
type Config struct {
	Name                 string `yaml:"name" validate:"required"`
	Kind                 string `yaml:"kind" validate:"required"`
	Region               string `yaml:"region" validate:"required"`
	Database             string `yaml:"database"`             // Optional: default database for queries
	OutputLocation       string `yaml:"outputLocation"`       // Optional: S3 location for query results (s3://bucket/path/)
	WorkGroup            string `yaml:"workGroup"`            // Optional: Athena workgroup for query execution
	EncryptionOption     string `yaml:"encryptionOption"`     // Optional: SSE_S3, SSE_KMS, CSE_KMS for result encryption
	KmsKey               string `yaml:"kmsKey"`               // Optional: KMS key ARN for encryption
	QueryResultsLocation string `yaml:"queryResultsLocation"` // Optional: S3 location for query results (alias for OutputLocation)
	AccessKeyID          string `yaml:"accessKeyId"`          // Optional: explicit credentials
	SecretAccessKey      string `yaml:"secretAccessKey"`      // Optional: explicit credentials
	SessionToken         string `yaml:"sessionToken"`         // Optional: session token
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initAthenaClient(ctx, tracer, r.Name, r.Region, r.AccessKeyID, r.SecretAccessKey, r.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create Athena client: %w", err)
	}

	// Verify the connection by listing databases
	_, err = client.ListDatabases(ctx, &athena.ListDatabasesInput{
		CatalogName: sourceutil.StringPtr("AwsDataCatalog"),
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

type Source struct {
	Config
	Client *athena.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// AthenaClient returns the underlying AWS Athena client for direct API access.
func (s *Source) AthenaClient() *athena.Client {
	return s.Client
}

// Close is not needed for this source because AWS SDK v2 clients manage
// their own connection pooling and cleanup automatically.

func initAthenaClient(ctx context.Context, tracer trace.Tracer, name, region, accessKeyID, secretAccessKey, sessionToken string) (*athena.Client, error) {
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

	// Create Athena client
	client := athena.NewFromConfig(cfg)

	return client, nil
}
