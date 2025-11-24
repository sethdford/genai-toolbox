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

package timestream

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlTimestream(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-timestream
kind: timestream
region: us-east-1`,
			wantErr: false,
			expected: Config{
				Name:   "test-timestream",
				Kind:   "timestream",
				Region: "us-east-1",
			},
		},
		{
			name: "valid configuration with database",
			yamlContent: `name: test-timestream
kind: timestream
region: us-west-2
database: myDatabase`,
			wantErr: false,
			expected: Config{
				Name:     "test-timestream",
				Kind:     "timestream",
				Region:   "us-west-2",
				Database: "myDatabase",
			},
		},
		{
			name: "valid configuration with different region",
			yamlContent: `name: prod-timestream
kind: timestream
region: eu-west-1
database: production_metrics`,
			wantErr: false,
			expected: Config{
				Name:     "prod-timestream",
				Kind:     "timestream",
				Region:   "eu-west-1",
				Database: "production_metrics",
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
			}
		})
	}
}

func TestFailParseFromYamlTimestream(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-timestream
kind: timestream
region: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-timestream
  kind: timestream
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

func TestSourceKindTimestream(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "timestream",
		Region: "us-east-1",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
