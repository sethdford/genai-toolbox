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

// Package qldb provides a source implementation for AWS QLDB ledger database.
//
// This source provides connectivity to Amazon Quantum Ledger Database (QLDB).
// QLDB is a fully managed ledger database with cryptographically verifiable transaction logs.
package qldb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/qldb"
	"github.com/aws/aws-sdk-go-v2/service/qldbsession"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "qldb"

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
	LedgerName      string `yaml:"ledgerName" validate:"required"`
	AccessKeyID     string `yaml:"accessKeyId"`     // Optional: explicit credentials
	SecretAccessKey string `yaml:"secretAccessKey"` // Optional: explicit credentials
	SessionToken    string `yaml:"sessionToken"`    // Optional: session token
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	qldbClient, sessionClient, err := initQLDBClients(ctx, tracer, r.Name, r.Region, r.AccessKeyID, r.SecretAccessKey, r.SessionToken)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create QLDB clients: %w", r.Name, SourceKind, err)
	}

	// Verify the connection by describing the ledger
	_, err = qldbClient.DescribeLedger(ctx, &qldb.DescribeLedgerInput{
		Name: &r.LedgerName,
	})
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to connect successfully: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config:        r,
		QLDBClient:    qldbClient,
		SessionClient: sessionClient,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	QLDBClient    *qldb.Client
	SessionClient *qldbsession.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// QLDBServiceClient returns the underlying AWS QLDB service client for direct API access.
func (s *Source) QLDBServiceClient() *qldb.Client {
	return s.QLDBClient
}

// QLDBSessionClient returns the underlying AWS QLDB session client for direct API access.
func (s *Source) QLDBSessionClient() *qldbsession.Client {
	return s.SessionClient
}

// Close is not needed for this source because AWS SDK v2 clients manage
// their own connection pooling and cleanup automatically.

func initQLDBClients(ctx context.Context, tracer trace.Tracer, name, region, accessKeyID, secretAccessKey, sessionToken string) (*qldb.Client, *qldbsession.Client, error) {
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

	// Create QLDB clients
	qldbClient := qldb.NewFromConfig(cfg)
	sessionClient := qldbsession.NewFromConfig(cfg)

	return qldbClient, sessionClient, nil
}
