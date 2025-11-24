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

// Package redshift provides a source implementation for AWS Redshift data warehouse.
//
// This source provides PostgreSQL-compatible connectivity to Amazon Redshift clusters.
// Connection pooling is configurable for different workload requirements.
package redshift

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	_ "github.com/lib/pq" // PostgreSQL driver (Redshift is PostgreSQL-compatible)
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "redshift"

// Default configuration constants
const (
	DefaultMaxOpenConns = 25          // Default maximum open connections
	DefaultMaxIdleConns = 5           // Default maximum idle connections
	DefaultConnMaxLifetime = time.Hour // Default connection maximum lifetime
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
	Name        string            `yaml:"name" validate:"required"`
	Kind        string            `yaml:"kind" validate:"required"`
	Host        string            `yaml:"host" validate:"required"` // e.g., mycluster.abc123.us-west-2.redshift.amazonaws.com
	Port        string            `yaml:"port" validate:"required"` // typically 5439
	User         string            `yaml:"user" validate:"required"`
	Password     string            `yaml:"password" validate:"required"`
	Database     string            `yaml:"database" validate:"required"`
	QueryParams  map[string]string `yaml:"queryParams"`
	MaxOpenConns int               `yaml:"maxOpenConns"` // Optional: max open connections (default 25)
	MaxIdleConns int               `yaml:"maxIdleConns"` // Optional: max idle connections (default 5)
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initRedshiftConnection(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.QueryParams, r.MaxOpenConns, r.MaxIdleConns)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to create connection: %w", r.Name, SourceKind, err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("source %q (%s): unable to connect successfully: %w", r.Name, SourceKind, err)
	}

	s := &Source{
		Config: r,
		DB:     db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	DB *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// RedshiftDB returns the underlying database connection for direct SQL operations.
func (s *Source) RedshiftDB() *sql.DB {
	return s.DB
}

// Close closes the database connection and releases resources.
func (s *Source) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	if s.DB != nil {
		return s.DB.Close()
	}
	return nil
}

func initRedshiftConnection(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname string, queryParams map[string]string, maxOpenConns, maxIdleConns int) (*sql.DB, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		userAgent = "genai-toolbox"
	}

	if queryParams == nil {
		queryParams = make(map[string]string)
	}
	if _, ok := queryParams["application_name"]; !ok {
		queryParams["application_name"] = userAgent
	}

	// Amazon Redshift uses PostgreSQL protocol
	// Connection string format: postgres://username:password@host:port/database?params
	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     dbname,
		RawQuery: convertParamMapToRawQuery(queryParams),
	}

	db, err := sql.Open("postgres", connURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to open connection: %w", err)
	}

	// Configure connection pool with defaults
	if maxOpenConns == 0 {
		maxOpenConns = DefaultMaxOpenConns
	}
	if maxIdleConns == 0 {
		maxIdleConns = DefaultMaxIdleConns
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(DefaultConnMaxLifetime)

	return db, nil
}

// convertParamMapToRawQuery safely encodes query parameters to prevent injection attacks.
// Uses url.Values for proper URL encoding instead of manual string concatenation.
func convertParamMapToRawQuery(queryParams map[string]string) string {
	values := url.Values{}
	for k, v := range queryParams {
		values.Add(k, v)
	}
	return values.Encode()
}
