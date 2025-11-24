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

// Package dynamodb provides a source implementation for AWS DynamoDB.
//
// This source supports both the AWS DynamoDB service and DynamoDB Local for testing.
// It can use either the default AWS credential chain or explicit credentials.
package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	sourceutil "github.com/googleapis/genai-toolbox/internal/sources/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "dynamodb"

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
	Endpoint        string `yaml:"endpoint"` // Optional: for DynamoDB Local
	AccessKeyID     string `yaml:"accessKeyId"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	SessionToken    string `yaml:"sessionToken"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initDynamoDBClient(ctx, tracer, r.Name, r.Region, r.Endpoint, r.AccessKeyID, r.SecretAccessKey, r.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("unable to create DynamoDB client: %w", err)
	}

	// Verify the connection by listing tables
	_, err = client.ListTables(ctx, &dynamodb.ListTablesInput{
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

type Source struct {
	Config
	Client *dynamodb.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// DynamoDBClient returns the underlying AWS DynamoDB client for direct API access.
func (s *Source) DynamoDBClient() *dynamodb.Client {
	return s.Client
}

// Close is not needed for this source because AWS SDK v2 clients manage
// their own connection pooling and cleanup automatically.

func initDynamoDBClient(ctx context.Context, tracer trace.Tracer, name, region, endpoint, accessKeyID, secretAccessKey, sessionToken string) (*dynamodb.Client, error) {
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

	// Create DynamoDB client options
	opts := []func(*dynamodb.Options){}

	// Add custom endpoint if specified (for DynamoDB Local)
	if endpoint != "" {
		opts = append(opts, func(o *dynamodb.Options) {
			o.BaseEndpoint = &endpoint
		})
	}

	// Create the DynamoDB client
	client := dynamodb.NewFromConfig(cfg, opts...)

	return client, nil
}
