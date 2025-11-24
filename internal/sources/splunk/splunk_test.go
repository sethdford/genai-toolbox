// Copyright 2025 Google LLC
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

package splunk_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/splunk"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlSplunk(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic token auth example",
			in: `
			sources:
				my-splunk-instance:
					kind: splunk
					host: splunk.example.com
					token: B5A79AAD-D822-46CC-80D1-819F80D7BFB0
			`,
			want: map[string]sources.SourceConfig{
				"my-splunk-instance": splunk.Config{
					Name:    "my-splunk-instance",
					Kind:    splunk.SourceKind,
					Host:    "splunk.example.com",
					Port:    8089,
					HECPort: 8088,
					Scheme:  "https",
					Token:   "B5A79AAD-D822-46CC-80D1-819F80D7BFB0",
					Timeout: "120s",
				},
			},
		},
		{
			desc: "username/password auth example",
			in: `
			sources:
				my-splunk-instance:
					kind: splunk
					host: splunk.example.com
					username: admin
					password: changeme
			`,
			want: map[string]sources.SourceConfig{
				"my-splunk-instance": splunk.Config{
					Name:     "my-splunk-instance",
					Kind:     splunk.SourceKind,
					Host:     "splunk.example.com",
					Port:     8089,
					HECPort:  8088,
					Scheme:   "https",
					Username: "admin",
					Password: "changeme",
					Timeout:  "120s",
				},
			},
		},
		{
			desc: "advanced example with HEC and custom ports",
			in: `
			sources:
				my-splunk-instance:
					kind: splunk
					host: splunk.example.com
					port: 8089
					hecPort: 8088
					scheme: https
					token: B5A79AAD-D822-46CC-80D1-819F80D7BFB0
					hecToken: 12345678-1234-1234-1234-123456789012
					timeout: 60s
					disableSslVerification: true
			`,
			want: map[string]sources.SourceConfig{
				"my-splunk-instance": splunk.Config{
					Name:                   "my-splunk-instance",
					Kind:                   splunk.SourceKind,
					Host:                   "splunk.example.com",
					Port:                   8089,
					HECPort:                8088,
					Scheme:                 "https",
					Token:                  "B5A79AAD-D822-46CC-80D1-819F80D7BFB0",
					HECToken:               "12345678-1234-1234-1234-123456789012",
					Timeout:                "60s",
					DisableSslVerification: true,
				},
			},
		},
		{
			desc: "minimal token example",
			in: `
			sources:
				splunk-dev:
					kind: splunk
					host: localhost
					token: test-token-123
			`,
			want: map[string]sources.SourceConfig{
				"splunk-dev": splunk.Config{
					Name:    "splunk-dev",
					Kind:    splunk.SourceKind,
					Host:    "localhost",
					Port:    8089,
					HECPort: 8088,
					Scheme:  "https",
					Token:   "test-token-123",
					Timeout: "120s",
				},
			},
		},
		{
			desc: "http scheme example",
			in: `
			sources:
				splunk-local:
					kind: splunk
					host: localhost
					scheme: http
					port: 8089
					username: admin
					password: password123
					disableSslVerification: true
			`,
			want: map[string]sources.SourceConfig{
				"splunk-local": splunk.Config{
					Name:                   "splunk-local",
					Kind:                   splunk.SourceKind,
					Host:                   "localhost",
					Port:                   8089,
					HECPort:                8088,
					Scheme:                 "http",
					Username:               "admin",
					Password:               "password123",
					Timeout:                "120s",
					DisableSslVerification: true,
				},
			},
		},
		{
			desc: "custom timeout example",
			in: `
			sources:
				splunk-prod:
					kind: splunk
					host: splunk.prod.example.com
					token: prod-token-xyz
					timeout: 300s
			`,
			want: map[string]sources.SourceConfig{
				"splunk-prod": splunk.Config{
					Name:    "splunk-prod",
					Kind:    splunk.SourceKind,
					Host:    "splunk.prod.example.com",
					Port:    8089,
					HECPort: 8088,
					Scheme:  "https",
					Token:   "prod-token-xyz",
					Timeout: "300s",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Sources) {
				t.Fatalf("incorrect parse: diff (-want +got):\n%s", cmp.Diff(tc.want, got.Sources))
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-splunk-instance:
					kind: splunk
					host: splunk.example.com
					token: test-token
					unknownField: value
			`,
			err: "unable to parse source \"my-splunk-instance\" as \"splunk\": [4:1] unknown field \"unknownField\"\n   1 | host: splunk.example.com\n   2 | kind: splunk\n   3 | token: test-token\n>  4 | unknownField: value\n       ^\n",
		},
		{
			desc: "missing kind field",
			in: `
			sources:
				my-splunk-instance:
					host: splunk.example.com
					token: test-token
			`,
			err: "missing 'kind' field for source \"my-splunk-instance\"",
		},
		{
			desc: "missing host field",
			in: `
			sources:
				my-splunk-instance:
					kind: splunk
					token: test-token
			`,
			err: "unable to parse source \"my-splunk-instance\" as \"splunk\": Key: 'Config.Host' Error:Field validation for 'Host' failed on the 'required' tag",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestSourceConfigKind(t *testing.T) {
	config := splunk.Config{
		Name: "test-splunk",
		Kind: splunk.SourceKind,
		Host: "localhost",
	}

	if got := config.SourceConfigKind(); got != splunk.SourceKind {
		t.Errorf("SourceConfigKind() = %q, want %q", got, splunk.SourceKind)
	}
}

// TestConfigValidation tests various configuration validation scenarios
func TestConfigValidation(t *testing.T) {
	tcs := []struct {
		desc    string
		yamlStr string
		wantErr bool
	}{
		{
			desc: "valid token auth",
			yamlStr: `
			sources:
				test:
					kind: splunk
					host: localhost
					token: test-token
			`,
			wantErr: false,
		},
		{
			desc: "valid username/password auth",
			yamlStr: `
			sources:
				test:
					kind: splunk
					host: localhost
					username: admin
					password: password
			`,
			wantErr: false,
		},
		{
			desc: "missing host",
			yamlStr: `
			sources:
				test:
					kind: splunk
					token: test-token
			`,
			wantErr: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			err := yaml.Unmarshal(testutils.FormatYaml(tc.yamlStr), &got)

			if tc.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestDefaultValues tests that default values are properly set
func TestDefaultValues(t *testing.T) {
	in := `
	sources:
		my-splunk:
			kind: splunk
			host: localhost
			token: test-token
	`

	got := struct {
		Sources server.SourceConfigs `yaml:"sources"`
	}{}

	err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	config, ok := got.Sources["my-splunk"].(splunk.Config)
	if !ok {
		t.Fatal("failed to cast to splunk.Config")
	}

	// Check default values
	if config.Port != 8089 {
		t.Errorf("Port = %d, want 8089", config.Port)
	}
	if config.HECPort != 8088 {
		t.Errorf("HECPort = %d, want 8088", config.HECPort)
	}
	if config.Scheme != "https" {
		t.Errorf("Scheme = %q, want \"https\"", config.Scheme)
	}
	if config.Timeout != "120s" {
		t.Errorf("Timeout = %q, want \"120s\"", config.Timeout)
	}
	if config.DisableSslVerification != false {
		t.Errorf("DisableSslVerification = %v, want false", config.DisableSslVerification)
	}
}

// TestMultipleSources tests parsing multiple Splunk sources
func TestMultipleSources(t *testing.T) {
	in := `
	sources:
		splunk-prod:
			kind: splunk
			host: splunk-prod.example.com
			token: prod-token
		splunk-dev:
			kind: splunk
			host: splunk-dev.example.com
			username: dev-user
			password: dev-pass
		splunk-test:
			kind: splunk
			host: localhost
			scheme: http
			port: 8089
			token: test-token
			disableSslVerification: true
	`

	got := struct {
		Sources server.SourceConfigs `yaml:"sources"`
	}{}

	err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify we have 3 sources
	if len(got.Sources) != 3 {
		t.Errorf("got %d sources, want 3", len(got.Sources))
	}

	// Verify each source
	expectedNames := []string{"splunk-prod", "splunk-dev", "splunk-test"}
	for _, name := range expectedNames {
		if _, ok := got.Sources[name]; !ok {
			t.Errorf("missing source %q", name)
		}
	}

	// Verify prod config
	prodConfig, ok := got.Sources["splunk-prod"].(splunk.Config)
	if !ok {
		t.Fatal("failed to cast splunk-prod to splunk.Config")
	}
	if prodConfig.Host != "splunk-prod.example.com" {
		t.Errorf("splunk-prod Host = %q, want \"splunk-prod.example.com\"", prodConfig.Host)
	}
	if prodConfig.Token != "prod-token" {
		t.Errorf("splunk-prod Token = %q, want \"prod-token\"", prodConfig.Token)
	}

	// Verify dev config
	devConfig, ok := got.Sources["splunk-dev"].(splunk.Config)
	if !ok {
		t.Fatal("failed to cast splunk-dev to splunk.Config")
	}
	if devConfig.Username != "dev-user" {
		t.Errorf("splunk-dev Username = %q, want \"dev-user\"", devConfig.Username)
	}
	if devConfig.Password != "dev-pass" {
		t.Errorf("splunk-dev Password = %q, want \"dev-pass\"", devConfig.Password)
	}

	// Verify test config
	testConfig, ok := got.Sources["splunk-test"].(splunk.Config)
	if !ok {
		t.Fatal("failed to cast splunk-test to splunk.Config")
	}
	if testConfig.Scheme != "http" {
		t.Errorf("splunk-test Scheme = %q, want \"http\"", testConfig.Scheme)
	}
	if !testConfig.DisableSslVerification {
		t.Error("splunk-test DisableSslVerification = false, want true")
	}
}

// TestHECConfiguration tests HEC-specific configuration
func TestHECConfiguration(t *testing.T) {
	in := `
	sources:
		splunk-hec:
			kind: splunk
			host: splunk.example.com
			token: api-token
			hecToken: hec-token-12345
			hecPort: 8088
	`

	got := struct {
		Sources server.SourceConfigs `yaml:"sources"`
	}{}

	err := yaml.Unmarshal(testutils.FormatYaml(in), &got)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	config, ok := got.Sources["splunk-hec"].(splunk.Config)
	if !ok {
		t.Fatal("failed to cast to splunk.Config")
	}

	if config.HECToken != "hec-token-12345" {
		t.Errorf("HECToken = %q, want \"hec-token-12345\"", config.HECToken)
	}
	if config.HECPort != 8088 {
		t.Errorf("HECPort = %d, want 8088", config.HECPort)
	}
}

// TestSchemeVariations tests different scheme configurations
func TestSchemeVariations(t *testing.T) {
	tcs := []struct {
		desc   string
		in     string
		scheme string
	}{
		{
			desc: "https scheme",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					scheme: https
					token: test-token
			`,
			scheme: "https",
		},
		{
			desc: "http scheme",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					scheme: http
					token: test-token
			`,
			scheme: "http",
		},
		{
			desc: "default scheme",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					token: test-token
			`,
			scheme: "https",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			config, ok := got.Sources["test"].(splunk.Config)
			if !ok {
				t.Fatal("failed to cast to splunk.Config")
			}

			if config.Scheme != tc.scheme {
				t.Errorf("Scheme = %q, want %q", config.Scheme, tc.scheme)
			}
		})
	}
}

// TestPortConfiguration tests various port configurations
func TestPortConfiguration(t *testing.T) {
	tcs := []struct {
		desc    string
		in      string
		port    int
		hecPort int
	}{
		{
			desc: "default ports",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					token: test-token
			`,
			port:    8089,
			hecPort: 8088,
		},
		{
			desc: "custom management port",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					port: 9089
					token: test-token
			`,
			port:    9089,
			hecPort: 8088,
		},
		{
			desc: "custom HEC port",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					hecPort: 9088
					token: test-token
			`,
			port:    8089,
			hecPort: 9088,
		},
		{
			desc: "custom both ports",
			in: `
			sources:
				test:
					kind: splunk
					host: localhost
					port: 9089
					hecPort: 9088
					token: test-token
			`,
			port:    9089,
			hecPort: 9088,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			config, ok := got.Sources["test"].(splunk.Config)
			if !ok {
				t.Fatal("failed to cast to splunk.Config")
			}

			if config.Port != tc.port {
				t.Errorf("Port = %d, want %d", config.Port, tc.port)
			}
			if config.HECPort != tc.hecPort {
				t.Errorf("HECPort = %d, want %d", config.HECPort, tc.hecPort)
			}
		})
	}
}
