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

package s3

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlS3(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-s3
kind: s3
region: us-east-1`,
			wantErr: false,
			expected: Config{
				Name:   "test-s3",
				Kind:   "s3",
				Region: "us-east-1",
			},
		},
		{
			name: "valid configuration with bucket",
			yamlContent: `name: test-s3
kind: s3
region: us-west-2
bucket: my-bucket`,
			wantErr: false,
			expected: Config{
				Name:   "test-s3",
				Kind:   "s3",
				Region: "us-west-2",
				Bucket: "my-bucket",
			},
		},
		{
			name: "valid configuration with endpoint",
			yamlContent: `name: test-s3
kind: s3
region: us-east-1
endpoint: http://localhost:9000
forcePathStyle: true`,
			wantErr: false,
			expected: Config{
				Name:           "test-s3",
				Kind:           "s3",
				Region:         "us-east-1",
				Endpoint:       "http://localhost:9000",
				ForcePathStyle: true,
			},
		},
		{
			name: "valid configuration with credentials",
			yamlContent: `name: test-s3
kind: s3
region: us-east-1
bucket: data-bucket
accessKeyId: AKIAIOSFODNN7EXAMPLE
secretAccessKey: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`,
			wantErr: false,
			expected: Config{
				Name:            "test-s3",
				Kind:            "s3",
				Region:          "us-east-1",
				Bucket:          "data-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
		{
			name: "valid configuration with MinIO",
			yamlContent: `name: minio-s3
kind: s3
region: us-east-1
endpoint: http://minio.example.com:9000
forcePathStyle: true
bucket: uploads
accessKeyId: minioadmin
secretAccessKey: minioadmin`,
			wantErr: false,
			expected: Config{
				Name:            "minio-s3",
				Kind:            "s3",
				Region:          "us-east-1",
				Endpoint:        "http://minio.example.com:9000",
				ForcePathStyle:  true,
				Bucket:          "uploads",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
		},
		{
			name: "valid configuration with all options",
			yamlContent: `name: prod-s3
kind: s3
region: eu-west-1
bucket: production-data
endpoint: https://s3.custom-domain.com
forcePathStyle: false
accessKeyId: AKIAEXAMPLE
secretAccessKey: secretexample`,
			wantErr: false,
			expected: Config{
				Name:            "prod-s3",
				Kind:            "s3",
				Region:          "eu-west-1",
				Bucket:          "production-data",
				Endpoint:        "https://s3.custom-domain.com",
				ForcePathStyle:  false,
				AccessKeyID:     "AKIAEXAMPLE",
				SecretAccessKey: "secretexample",
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
				if tt.expected.Bucket != "" {
					assert.Equal(t, tt.expected.Bucket, config.(Config).Bucket)
				}
				if tt.expected.Endpoint != "" {
					assert.Equal(t, tt.expected.Endpoint, config.(Config).Endpoint)
				}
				assert.Equal(t, tt.expected.ForcePathStyle, config.(Config).ForcePathStyle)
				if tt.expected.AccessKeyID != "" {
					assert.Equal(t, tt.expected.AccessKeyID, config.(Config).AccessKeyID)
				}
				if tt.expected.SecretAccessKey != "" {
					assert.Equal(t, tt.expected.SecretAccessKey, config.(Config).SecretAccessKey)
				}
			}
		})
	}
}

func TestFailParseFromYamlS3(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-s3
kind: s3
region: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-s3
  kind: s3
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

func TestSourceKindS3(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "s3",
		Region: "us-east-1",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
