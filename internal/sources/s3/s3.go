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

// Package s3 provides a source implementation for AWS S3 and S3-compatible storage.
//
// This source supports both AWS S3 and S3-compatible services like MinIO.
// It supports both virtual-hosted-style and path-style URL addressing.
package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "s3"

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
	Bucket          string `yaml:"bucket"`          // Optional: default bucket
	Endpoint        string `yaml:"endpoint"`        // Optional: for S3-compatible services
	ForcePathStyle  bool   `yaml:"forcePathStyle"`  // Optional: use path-style addressing
	AccessKeyID     string `yaml:"accessKeyId"`     // Optional: for explicit credentials
	SecretAccessKey string `yaml:"secretAccessKey"` // Optional: for explicit credentials
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initS3Client(ctx, tracer, r.Name, r.Region, r.Endpoint, r.ForcePathStyle, r.AccessKeyID, r.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create S3 client: %w", r.Name, SourceKind, err)
	}

	// Verify the connection by listing buckets
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
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
	Client *s3.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// S3Client returns the underlying AWS S3 client for direct API access.
func (s *Source) S3Client() *s3.Client {
	return s.Client
}

// Close is not needed for this source because AWS SDK v2 clients manage
// their own connection pooling and cleanup automatically.

func initS3Client(ctx context.Context, tracer trace.Tracer, name, region, endpoint string, forcePathStyle bool, accessKeyID, secretAccessKey string) (*s3.Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build AWS config load options
	configOpts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	// Use explicit credentials if provided (same pattern as DynamoDB)
	if accessKeyID != "" && secretAccessKey != "" {
		configOpts = append(configOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		))
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	// Create S3 client options
	opts := []func(*s3.Options){}

	// Apply path style regardless of endpoint
	if forcePathStyle {
		opts = append(opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	// Add custom endpoint separately if specified (for S3-compatible services like MinIO)
	if endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = &endpoint
		})
	}

	// Create the S3 client
	client := s3.NewFromConfig(cfg, opts...)

	return client, nil
}
