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

package dynamodb

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestDynamoDBConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-dynamodb
kind: dynamodb
region: us-east-1`,
			wantErr: false,
			expected: Config{
				Name:   "test-dynamodb",
				Kind:   "dynamodb",
				Region: "us-east-1",
			},
		},
		{
			name: "valid configuration with endpoint",
			yamlContent: `name: test-dynamodb
kind: dynamodb
region: us-west-2
endpoint: http://localhost:8000`,
			wantErr: false,
			expected: Config{
				Name:     "test-dynamodb",
				Kind:     "dynamodb",
				Region:   "us-west-2",
				Endpoint: "http://localhost:8000",
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
				assert.Equal(t, tt.expected.Region, config.(Config).Region)
			}
		})
	}
}

func TestSourceKind(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "dynamodb",
		Region: "us-east-1",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
