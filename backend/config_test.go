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
