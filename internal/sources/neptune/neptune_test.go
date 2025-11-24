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

package neptune

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlNeptune(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-neptune
kind: neptune
endpoint: wss://my-neptune.cluster-abc123.us-east-1.neptune.amazonaws.com:8182/gremlin`,
			wantErr: false,
			expected: Config{
				Name:     "test-neptune",
				Kind:     "neptune",
				Endpoint: "wss://my-neptune.cluster-abc123.us-east-1.neptune.amazonaws.com:8182/gremlin",
				UseIAM:   false,
			},
		},
		{
			name: "valid configuration with IAM",
			yamlContent: `name: test-neptune
kind: neptune
endpoint: wss://my-neptune.cluster-abc123.us-east-1.neptune.amazonaws.com:8182/gremlin
useIAM: true`,
			wantErr: false,
			expected: Config{
				Name:     "test-neptune",
				Kind:     "neptune",
				Endpoint: "wss://my-neptune.cluster-abc123.us-east-1.neptune.amazonaws.com:8182/gremlin",
				UseIAM:   true,
			},
		},
		{
			name: "valid configuration with localhost",
			yamlContent: `name: local-neptune
kind: neptune
endpoint: ws://localhost:8182/gremlin`,
			wantErr: false,
			expected: Config{
				Name:     "local-neptune",
				Kind:     "neptune",
				Endpoint: "ws://localhost:8182/gremlin",
				UseIAM:   false,
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
				assert.Equal(t, tt.expected.Endpoint, config.(Config).Endpoint)
				assert.Equal(t, tt.expected.UseIAM, config.(Config).UseIAM)
			}
		})
	}
}

func TestFailParseFromYamlNeptune(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-neptune
kind: neptune
endpoint: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-neptune
  kind: neptune
    endpoint: wss://localhost:8182/gremlin`,
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

func TestSourceKindNeptune(t *testing.T) {
	config := Config{
		Name:     "test",
		Kind:     "neptune",
		Endpoint: "ws://localhost:8182/gremlin",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
