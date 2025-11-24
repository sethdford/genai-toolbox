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

package athena

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlAthena(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-athena
kind: athena
region: us-east-1`,
			wantErr: false,
			expected: Config{
				Name:   "test-athena",
				Kind:   "athena",
				Region: "us-east-1",
			},
		},
		{
			name: "valid configuration with database",
			yamlContent: `name: test-athena
kind: athena
region: us-west-2
database: my_database`,
			wantErr: false,
			expected: Config{
				Name:     "test-athena",
				Kind:     "athena",
				Region:   "us-west-2",
				Database: "my_database",
			},
		},
		{
			name: "valid configuration with output location",
			yamlContent: `name: test-athena
kind: athena
region: us-east-1
database: analytics
outputLocation: s3://my-bucket/athena-results/`,
			wantErr: false,
			expected: Config{
				Name:           "test-athena",
				Kind:           "athena",
				Region:         "us-east-1",
				Database:       "analytics",
				OutputLocation: "s3://my-bucket/athena-results/",
			},
		},
		{
			name: "valid configuration with workgroup",
			yamlContent: `name: test-athena
kind: athena
region: eu-west-1
workGroup: primary`,
			wantErr: false,
			expected: Config{
				Name:      "test-athena",
				Kind:      "athena",
				Region:    "eu-west-1",
				WorkGroup: "primary",
			},
		},
		{
			name: "valid configuration with encryption",
			yamlContent: `name: test-athena
kind: athena
region: us-east-1
encryptionOption: SSE_KMS
kmsKey: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012`,
			wantErr: false,
			expected: Config{
				Name:             "test-athena",
				Kind:             "athena",
				Region:           "us-east-1",
				EncryptionOption: "SSE_KMS",
				KmsKey:           "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			},
		},
		{
			name: "valid configuration with all options",
			yamlContent: `name: prod-athena
kind: athena
region: us-west-2
database: production
outputLocation: s3://prod-bucket/results/
workGroup: production
encryptionOption: SSE_S3
queryResultsLocation: s3://prod-bucket/query-results/`,
			wantErr: false,
			expected: Config{
				Name:                 "prod-athena",
				Kind:                 "athena",
				Region:               "us-west-2",
				Database:             "production",
				OutputLocation:       "s3://prod-bucket/results/",
				WorkGroup:            "production",
				EncryptionOption:     "SSE_S3",
				QueryResultsLocation: "s3://prod-bucket/query-results/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlContent)))
			config, err := newConfig(context.Background(), tt.expected.Name, decoder)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, config.(Config).Name)
				assert.Equal(t, tt.expected.Kind, config.(Config).Kind)
				assert.Equal(t, tt.expected.Region, config.(Config).Region)
				if tt.expected.Database != "" {
					assert.Equal(t, tt.expected.Database, config.(Config).Database)
				}
				if tt.expected.OutputLocation != "" {
					assert.Equal(t, tt.expected.OutputLocation, config.(Config).OutputLocation)
				}
				if tt.expected.WorkGroup != "" {
					assert.Equal(t, tt.expected.WorkGroup, config.(Config).WorkGroup)
				}
				if tt.expected.EncryptionOption != "" {
					assert.Equal(t, tt.expected.EncryptionOption, config.(Config).EncryptionOption)
				}
				if tt.expected.KmsKey != "" {
					assert.Equal(t, tt.expected.KmsKey, config.(Config).KmsKey)
				}
				if tt.expected.QueryResultsLocation != "" {
					assert.Equal(t, tt.expected.QueryResultsLocation, config.(Config).QueryResultsLocation)
				}
			}
		})
	}
}

func TestFailParseFromYamlAthena(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-athena
kind: athena
region: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-athena
  kind: athena
    region: us-east-1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(tt.yamlContent)))
			_, err := newConfig(context.Background(), "test", decoder)
			assert.Error(t, err)
		})
	}
}

func TestSourceKindAthena(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "athena",
		Region: "us-east-1",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
