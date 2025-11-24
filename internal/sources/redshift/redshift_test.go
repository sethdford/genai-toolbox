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

package redshift

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestRedshiftConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-redshift
kind: redshift
host: mycluster.abc123.us-west-2.redshift.amazonaws.com
port: "5439"
user: admin
password: mypassword
database: mydb`,
			wantErr: false,
			expected: Config{
				Name:     "test-redshift",
				Kind:     "redshift",
				Host:     "mycluster.abc123.us-west-2.redshift.amazonaws.com",
				Port:     "5439",
				User:     "admin",
				Password: "mypassword",
				Database: "mydb",
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
				assert.Equal(t, tt.expected.Host, config.(Config).Host)
			}
		})
	}
}

func TestSourceKind(t *testing.T) {
	config := Config{
		Name:     "test",
		Kind:     "redshift",
		Host:     "test.redshift.amazonaws.com",
		Port:     "5439",
		User:     "user",
		Password: "pass",
		Database: "db",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
