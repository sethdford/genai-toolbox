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

package honeycomb

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestHoneycombConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		expected    Config
	}{
		{
			name: "valid configuration",
			yamlContent: `name: test-honeycomb
kind: honeycomb
apiKey: hcxik_test123456789`,
			wantErr: false,
			expected: Config{
				Name:   "test-honeycomb",
				Kind:   "honeycomb",
				APIKey: "hcxik_test123456789",
			},
		},
		{
			name: "valid configuration with dataset",
			yamlContent: `name: test-honeycomb
kind: honeycomb
apiKey: hcxik_test123456789
dataset: my-dataset`,
			wantErr: false,
			expected: Config{
				Name:    "test-honeycomb",
				Kind:    "honeycomb",
				APIKey:  "hcxik_test123456789",
				Dataset: "my-dataset",
			},
		},
		{
			name: "valid configuration with environment",
			yamlContent: `name: test-honeycomb
kind: honeycomb
apiKey: hcxik_test123456789
dataset: my-dataset
environment: production`,
			wantErr: false,
			expected: Config{
				Name:        "test-honeycomb",
				Kind:        "honeycomb",
				APIKey:      "hcxik_test123456789",
				Dataset:     "my-dataset",
				Environment: "production",
			},
		},
		{
			name: "valid configuration with custom base URL",
			yamlContent: `name: test-honeycomb
kind: honeycomb
apiKey: hcxik_test123456789
baseUrl: https://custom-honeycomb.example.com`,
			wantErr: false,
			expected: Config{
				Name:    "test-honeycomb",
				Kind:    "honeycomb",
				APIKey:  "hcxik_test123456789",
				BaseURL: "https://custom-honeycomb.example.com",
			},
		},
		{
			name: "valid configuration with timeout",
			yamlContent: `name: test-honeycomb
kind: honeycomb
apiKey: hcxik_test123456789
timeout: 60`,
			wantErr: false,
			expected: Config{
				Name:    "test-honeycomb",
				Kind:    "honeycomb",
				APIKey:  "hcxik_test123456789",
				Timeout: 60,
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
				assert.Equal(t, tt.expected.APIKey, config.(Config).APIKey)
				assert.Equal(t, tt.expected.Dataset, config.(Config).Dataset)
				assert.Equal(t, tt.expected.Environment, config.(Config).Environment)
				if tt.expected.BaseURL != "" {
					assert.Equal(t, tt.expected.BaseURL, config.(Config).BaseURL)
				}
				if tt.expected.Timeout != 0 {
					assert.Equal(t, tt.expected.Timeout, config.(Config).Timeout)
				}
			}
		})
	}
}

func TestSourceKind(t *testing.T) {
	config := Config{
		Name:   "test",
		Kind:   "honeycomb",
		APIKey: "test-key",
	}
	assert.Equal(t, SourceKind, config.SourceConfigKind())

	source := Source{Config: config}
	assert.Equal(t, SourceKind, source.SourceKind())
}

func TestInitHoneycombClient(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		baseURL    string
		timeout    int
		wantErr    bool
		wantURL    string
		wantAPIKey string
	}{
		{
			name:       "valid client with defaults",
			apiKey:     "hcxik_test123456789",
			baseURL:    "",
			timeout:    0,
			wantErr:    false,
			wantURL:    "https://api.honeycomb.io",
			wantAPIKey: "hcxik_test123456789",
		},
		{
			name:       "valid client with custom base URL",
			apiKey:     "hcxik_test123456789",
			baseURL:    "https://custom.honeycomb.io",
			timeout:    60,
			wantErr:    false,
			wantURL:    "https://custom.honeycomb.io",
			wantAPIKey: "hcxik_test123456789",
		},
		{
			name:    "missing API key",
			apiKey:  "",
			baseURL: "",
			timeout: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tracer := noop.NewTracerProvider().Tracer("test")

			client, err := initHoneycombClient(ctx, tracer, "test", tt.apiKey, tt.baseURL, tt.timeout)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.wantURL, client.BaseURL)
				assert.Equal(t, tt.wantAPIKey, client.APIKey)
				assert.NotNil(t, client.HTTPClient)
			}
		})
	}
}

func TestListDatasets(t *testing.T) {
	expectedDatasets := []Dataset{
		{
			Name:        "test-dataset-1",
			Slug:        "test-dataset-1",
			Description: "Test dataset 1",
			Created:     "2024-01-01T00:00:00Z",
		},
		{
			Name:        "test-dataset-2",
			Slug:        "test-dataset-2",
			Description: "Test dataset 2",
			Created:     "2024-01-02T00:00:00Z",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/1/datasets", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("X-Honeycomb-Team"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedDatasets)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	datasets, err := client.ListDatasets(ctx)

	assert.NoError(t, err)
	assert.Len(t, datasets, 2)
	assert.Equal(t, expectedDatasets[0].Name, datasets[0].Name)
	assert.Equal(t, expectedDatasets[1].Name, datasets[1].Name)
}

func TestCreateQuery(t *testing.T) {
	expectedQuery := Query{
		ID: "test-query-id",
		QuerySpec: QuerySpec{
			Calculations: []Calculation{
				{Op: "COUNT"},
			},
			TimeRange: 3600,
		},
		Created: "2024-01-01T00:00:00Z",
		Updated: "2024-01-01T00:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/1/queries/test-dataset", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("X-Honeycomb-Team"))

		var spec QuerySpec
		err := json.NewDecoder(r.Body).Decode(&spec)
		assert.NoError(t, err)
		assert.Len(t, spec.Calculations, 1)
		assert.Equal(t, "COUNT", spec.Calculations[0].Op)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(expectedQuery)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	spec := QuerySpec{
		Calculations: []Calculation{
			{Op: "COUNT"},
		},
		TimeRange: 3600,
	}

	query, err := client.CreateQuery(ctx, "test-dataset", spec)

	assert.NoError(t, err)
	assert.NotNil(t, query)
	assert.Equal(t, expectedQuery.ID, query.ID)
	assert.Len(t, query.QuerySpec.Calculations, 1)
}

func TestExecuteQuery(t *testing.T) {
	expectedResult := QueryResult{
		ID:       "test-result-id",
		QueryID:  "test-query-id",
		Complete: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/1/query_results/test-dataset", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("X-Honeycomb-Team"))

		var request map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, err)
		assert.Equal(t, "test-query-id", request["query_id"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResult)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	result, err := client.ExecuteQuery(ctx, "test-dataset", "test-query-id")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult.ID, result.ID)
	assert.Equal(t, expectedResult.QueryID, result.QueryID)
	assert.False(t, result.Complete)
}

func TestGetQueryResult(t *testing.T) {
	expectedResult := QueryResult{
		ID:       "test-result-id",
		QueryID:  "test-query-id",
		Complete: true,
		Data: []map[string]interface{}{
			{
				"series": map[string]interface{}{
					"time": 1704067200000,
				},
				"data": []interface{}{
					map[string]interface{}{
						"COUNT": 100.0,
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/1/query_results/test-dataset/test-result-id", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "test-api-key", r.Header.Get("X-Honeycomb-Team"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResult)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	result, err := client.GetQueryResult(ctx, "test-dataset", "test-result-id")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedResult.ID, result.ID)
	assert.True(t, result.Complete)
	assert.Len(t, result.Data, 1)
}

func TestPollQueryResult(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		result := QueryResult{
			ID:       "test-result-id",
			QueryID:  "test-query-id",
			Complete: callCount >= 3, // Complete on the 3rd call
			Data:     []map[string]interface{}{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	result, err := client.PollQueryResult(ctx, "test-dataset", "test-result-id", 5)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Complete)
	assert.GreaterOrEqual(t, callCount, 3)
}

func TestPollQueryResultTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := QueryResult{
			ID:       "test-result-id",
			QueryID:  "test-query-id",
			Complete: false, // Never complete
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	result, err := client.PollQueryResult(ctx, "test-dataset", "test-result-id", 2)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "did not complete")
}

func TestPollQueryResultError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := QueryResult{
			ID:       "test-result-id",
			QueryID:  "test-query-id",
			Complete: false,
			Error:    "query execution failed",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}))
	defer server.Close()

	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	ctx := context.Background()
	result, err := client.PollQueryResult(ctx, "test-dataset", "test-result-id", 5)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "query failed")
}

func TestAPIErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid API key"}`,
			expectedErrMsg: "API request failed with status 401",
		},
		{
			name:           "not found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "Dataset not found"}`,
			expectedErrMsg: "API request failed with status 404",
		},
		{
			name:           "rate limit",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"error": "Rate limit exceeded"}`,
			expectedErrMsg: "API request failed with status 429",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				APIKey:     "test-api-key",
				BaseURL:    server.URL,
				HTTPClient: server.Client(),
			}

			ctx := context.Background()
			_, err := client.ListDatasets(ctx)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

func TestToConfig(t *testing.T) {
	config := Config{
		Name:        "test",
		Kind:        "honeycomb",
		APIKey:      "test-key",
		Dataset:     "test-dataset",
		Environment: "production",
	}

	source := Source{Config: config}
	retrievedConfig := source.ToConfig()

	assert.Equal(t, config.Name, retrievedConfig.(Config).Name)
	assert.Equal(t, config.APIKey, retrievedConfig.(Config).APIKey)
	assert.Equal(t, config.Dataset, retrievedConfig.(Config).Dataset)
	assert.Equal(t, config.Environment, retrievedConfig.(Config).Environment)
}

func TestHoneycombClientAccessor(t *testing.T) {
	client := &Client{
		APIKey:     "test-api-key",
		BaseURL:    "https://api.honeycomb.io",
		HTTPClient: &http.Client{},
	}

	source := Source{
		Config: Config{
			Name:   "test",
			Kind:   "honeycomb",
			APIKey: "test-api-key",
		},
		Client: client,
	}

	retrievedClient := source.HoneycombClient()
	assert.Equal(t, client, retrievedClient)
	assert.Equal(t, "test-api-key", retrievedClient.APIKey)
}
