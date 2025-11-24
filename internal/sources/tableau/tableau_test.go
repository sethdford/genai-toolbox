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

package tableau

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

func TestParseFromYamlTableau(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration with username and password",
			yamlContent: `name: test-tableau
kind: tableau
serverUrl: https://tableau.example.com
username: admin
password: secret123`,
			wantErr: false,
			expected: Config{
				Name:      "test-tableau",
				Kind:      "tableau",
				ServerURL: "https://tableau.example.com",
				Username:  "admin",
				Password:  "secret123",
			},
		},
		{
			name: "valid configuration with PAT",
			yamlContent: `name: test-tableau
kind: tableau
serverUrl: https://tableau.example.com
personalAccessTokenName: my-token
personalAccessTokenSecret: token-secret-value`,
			wantErr: false,
			expected: Config{
				Name:                      "test-tableau",
				Kind:                      "tableau",
				ServerURL:                 "https://tableau.example.com",
				PersonalAccessTokenName:   "my-token",
				PersonalAccessTokenSecret: "token-secret-value",
			},
		},
		{
			name: "valid configuration with site name",
			yamlContent: `name: test-tableau
kind: tableau
serverUrl: https://tableau.example.com
siteName: marketing
username: admin
password: secret123`,
			wantErr: false,
			expected: Config{
				Name:      "test-tableau",
				Kind:      "tableau",
				ServerURL: "https://tableau.example.com",
				SiteName:  "marketing",
				Username:  "admin",
				Password:  "secret123",
			},
		},
		{
			name: "valid configuration with API version",
			yamlContent: `name: test-tableau
kind: tableau
serverUrl: https://tableau.example.com
username: admin
password: secret123
apiVersion: "3.19"`,
			wantErr: false,
			expected: Config{
				Name:       "test-tableau",
				Kind:       "tableau",
				ServerURL:  "https://tableau.example.com",
				Username:   "admin",
				Password:   "secret123",
				APIVersion: "3.19",
			},
		},
		{
			name: "valid configuration with all options",
			yamlContent: `name: prod-tableau
kind: tableau
serverUrl: https://tableau-prod.example.com
siteName: production
personalAccessTokenName: prod-token
personalAccessTokenSecret: prod-secret
apiVersion: "3.27"`,
			wantErr: false,
			expected: Config{
				Name:                      "prod-tableau",
				Kind:                      "tableau",
				ServerURL:                 "https://tableau-prod.example.com",
				SiteName:                  "production",
				PersonalAccessTokenName:   "prod-token",
				PersonalAccessTokenSecret: "prod-secret",
				APIVersion:                "3.27",
			},
		},
		{
			name: "valid configuration with localhost",
			yamlContent: `name: local-tableau
kind: tableau
serverUrl: http://localhost:8080
username: testuser
password: testpass`,
			wantErr: false,
			expected: Config{
				Name:      "local-tableau",
				Kind:      "tableau",
				ServerURL: "http://localhost:8080",
				Username:  "testuser",
				Password:  "testpass",
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
				assert.Equal(t, tt.expected.ServerURL, config.(Config).ServerURL)
				if tt.expected.SiteName != "" {
					assert.Equal(t, tt.expected.SiteName, config.(Config).SiteName)
				}
				if tt.expected.Username != "" {
					assert.Equal(t, tt.expected.Username, config.(Config).Username)
				}
				if tt.expected.Password != "" {
					assert.Equal(t, tt.expected.Password, config.(Config).Password)
				}
				if tt.expected.PersonalAccessTokenName != "" {
					assert.Equal(t, tt.expected.PersonalAccessTokenName, config.(Config).PersonalAccessTokenName)
				}
				if tt.expected.PersonalAccessTokenSecret != "" {
					assert.Equal(t, tt.expected.PersonalAccessTokenSecret, config.(Config).PersonalAccessTokenSecret)
				}
				if tt.expected.APIVersion != "" {
					assert.Equal(t, tt.expected.APIVersion, config.(Config).APIVersion)
				}
			}
		})
	}
}

func TestFailParseFromYamlTableau(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
	}{
		{
			name: "invalid yaml syntax",
			yamlContent: `name: test-tableau
kind: tableau
serverUrl: [invalid yaml syntax`,
		},
		{
			name: "malformed yaml structure",
			yamlContent: `name: test-tableau
  kind: tableau
    serverUrl: https://tableau.example.com`,
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

func TestSourceKindTableau(t *testing.T) {
	config := Config{
		Name:      "test",
		Kind:      "tableau",
		ServerURL: "https://tableau.example.com",
		Username:  "testuser",
		Password:  "testpass",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}
