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

package qldb

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlQLDB(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-qldb
kind: qldb
region: us-east-1
ledgerName: myLedger`,
			wantErr: false,
			expected: Config{
				Name:       "test-qldb",
				Kind:       "qldb",
				Region:     "us-east-1",
				LedgerName: "myLedger",
			},
		},
		{
			name: "valid configuration with different region",
			yamlContent: `name: prod-qldb
kind: qldb
region: eu-west-1
ledgerName: production-ledger`,
			wantErr: false,
			expected: Config{
				Name:       "prod-qldb",
				Kind:       "qldb",
				Region:     "eu-west-1",
				LedgerName: "production-ledger",
			},
		},
		{
			name: "valid configuration in us-west-2",
			yamlContent: `name: west-qldb
kind: qldb
region: us-west-2
ledgerName: vehicle-registration`,
			wantErr: false,
			expected: Config{
				Name:       "west-qldb",
				Kind:       "qldb",
				Region:     "us-west-2",
				LedgerName: "vehicle-registration",
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
				assert.Equal(t, tt.expected.LedgerName, config.(Config).LedgerName)
			}
		})
	}
}

func TestFailParseFromYamlQLDB(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-qldb
kind: qldb
region: [invalid yaml syntax
ledgerName: myLedger`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-qldb
  kind: qldb
    region: us-east-1
      ledgerName: myLedger`,
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

func TestSourceKindQLDB(t *testing.T) {
	config := Config{
		Name:       "test",
		Kind:       "qldb",
		Region:     "us-east-1",
		LedgerName: "test-ledger",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
