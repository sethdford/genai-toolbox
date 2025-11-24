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

package documentdb

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlDocumentDB(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-documentdb
kind: documentdb
uri: mongodb://admin:password@docdb-cluster.cluster-abc123.us-east-1.docdb.amazonaws.com:27017`,
			wantErr: false,
			expected: Config{
				Name: "test-documentdb",
				Kind: "documentdb",
				Uri:  "mongodb://admin:password@docdb-cluster.cluster-abc123.us-east-1.docdb.amazonaws.com:27017",
			},
		},
		{
			name: "valid configuration with TLS CA file",
			yamlContent: `name: test-documentdb
kind: documentdb
uri: mongodb://admin:password@docdb-cluster.cluster-abc123.us-east-1.docdb.amazonaws.com:27017
tlsCAFile: /path/to/ca-cert.pem`,
			wantErr: false,
			expected: Config{
				Name:      "test-documentdb",
				Kind:      "documentdb",
				Uri:       "mongodb://admin:password@docdb-cluster.cluster-abc123.us-east-1.docdb.amazonaws.com:27017",
				TLSCAFile: "/path/to/ca-cert.pem",
			},
		},
		{
			name: "valid configuration with localhost",
			yamlContent: `name: local-documentdb
kind: documentdb
uri: mongodb://localhost:27017`,
			wantErr: false,
			expected: Config{
				Name: "local-documentdb",
				Kind: "documentdb",
				Uri:  "mongodb://localhost:27017",
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
				assert.Equal(t, tt.expected.Uri, config.(Config).Uri)
				if tt.expected.TLSCAFile != "" {
					assert.Equal(t, tt.expected.TLSCAFile, config.(Config).TLSCAFile)
				}
			}
		})
	}
}

func TestFailParseFromYamlDocumentDB(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-documentdb
kind: documentdb
uri: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-documentdb
  kind: documentdb
    uri: mongodb://localhost:27017`,
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

func TestSourceKindDocumentDB(t *testing.T) {
	config := Config{
		Name: "test",
		Kind: "documentdb",
		Uri:  "mongodb://localhost:27017",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
