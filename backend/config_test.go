package main

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestDefaultConfigFromEnv_UsesDockerHosts(t *testing.T) {
	t.Setenv("DOCKER_HOSTS", "unix:///var/run/docker.sock,tcp://127.0.0.1:2375")

	cfg, err := defaultConfigFromEnv()
	if err != nil {
		t.Fatalf("defaultConfigFromEnv error: %v", err)
	}
	if len(cfg.Hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(cfg.Hosts))
	}
	if cfg.Hosts[0].Name != "host-1" || cfg.Hosts[1].Name != "host-2" {
		t.Fatalf("unexpected host names: %+v", cfg.Hosts)
	}
}

func TestWriteConfigYAML_RoundTrip(t *testing.T) {
	cfg := Config{
		Hosts: []Host{{Name: "h1", Endpoint: "unix:///var/run/docker.sock"}},
		ProxyMappings: map[string]string{
			"example": "http://localhost:8080",
		},
		Defaults: Defaults{BaseDomain: "local", Scheme: "http"},
	}

	var buf bytes.Buffer
	if err := writeConfigYAML(&buf, cfg); err != nil {
		t.Fatalf("writeConfigYAML error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Fatalf("expected trailing newline")
	}

	var got Config
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("yaml.Unmarshal error: %v", err)
	}
	if len(got.Hosts) != 1 || got.Hosts[0].Name != "h1" {
		t.Fatalf("unexpected round-trip config: %+v", got)
	}
}

func TestParseProxyMapping_ValidFormats(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHost  string
		wantCont  string
		wantPort  int
		wantError bool
	}{
		{"with port", "homelab/api:8080", "homelab", "api", 8080, false},
		{"without port", "homelab/web", "homelab", "web", 0, false},
		{"complex name with port", "remote-server/my-container:3000", "remote-server", "my-container", 3000, false},
		{"standard ports", "host/container:80", "host", "container", 80, false},
		{"https port", "host/container:443", "host", "container", 443, false},
		{"max port", "host/container:65535", "host", "container", 65535, false},
		{"empty string", "", "", "", 0, true},
		{"missing slash", "hostcontainer:8080", "", "", 0, true},
		{"missing host", "/container:8080", "", "", 0, true},
		{"missing container", "host/:8080", "", "", 0, true},
		{"invalid port negative", "host/container:-1", "", "", 0, true},
		{"invalid port zero", "host/container:0", "", "", 0, true},
		{"invalid port too large", "host/container:65536", "", "", 0, true},
		{"invalid port non-numeric", "host/container:abc", "", "", 0, true},
		{"port with colon but empty", "host/container:", "host", "container", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProxyMapping(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Host != tt.wantHost {
				t.Errorf("Host: got %q, want %q", got.Host, tt.wantHost)
			}
			if got.Container != tt.wantCont {
				t.Errorf("Container: got %q, want %q", got.Container, tt.wantCont)
			}
			if got.Port != tt.wantPort {
				t.Errorf("Port: got %d, want %d", got.Port, tt.wantPort)
			}
		})
	}
}

func TestParseProxyMappings_SkipsInvalid(t *testing.T) {
	input := map[string]string{
		"valid1.com":   "host1/container1:8080",
		"valid2.com":   "host2/container2",
		"invalid1.com": "invalid-format",
		"invalid2.com": "host/container:99999",
	}

	result := parseProxyMappings(input)

	if len(result) != 2 {
		t.Fatalf("expected 2 valid mappings, got %d", len(result))
	}

	if _, ok := result["valid1.com"]; !ok {
		t.Error("expected valid1.com in result")
	}
	if _, ok := result["valid2.com"]; !ok {
		t.Error("expected valid2.com in result")
	}
	if _, ok := result["invalid1.com"]; ok {
		t.Error("invalid1.com should not be in result")
	}
	if _, ok := result["invalid2.com"]; ok {
		t.Error("invalid2.com should not be in result")
	}
}
