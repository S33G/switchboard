package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

func loadConfigFromEnv() (Config, string, error) {
	path := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, "", err
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, "", err
		}
		cfg.ProxyMappings = ensureMapping(cfg.ProxyMappings)
		cfg.HostAddresses = ensureMapping(cfg.HostAddresses)
		cfg.Hosts = ensureHostNames(cfg.Hosts)
		cfg.ParsedMappings = parseProxyMappings(cfg.ProxyMappings)
		if len(cfg.Hosts) == 0 {
			return Config{}, "", fmt.Errorf("no hosts configured")
		}
		return cfg, path, nil
	}

	// Try default_config.yaml
	defaultPath := "default_config.yaml"
	if data, err := os.ReadFile(defaultPath); err == nil {
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err == nil {
			cfg.ProxyMappings = ensureMapping(cfg.ProxyMappings)
			cfg.HostAddresses = ensureMapping(cfg.HostAddresses)
			cfg.Hosts = ensureHostNames(cfg.Hosts)
			cfg.ParsedMappings = parseProxyMappings(cfg.ProxyMappings)
			if len(cfg.Hosts) > 0 {
				return cfg, defaultPath, nil
			}
		}
	}

	cfg, err := defaultConfigFromEnv()
	return cfg, "DOCKER_HOSTS env", err
}

func defaultConfigFromEnv() (Config, error) {
	value := strings.TrimSpace(os.Getenv("DOCKER_HOSTS"))
	if value == "" {
		value = "unix:///var/run/docker.sock"
	}
	hosts := parseHosts(value)
	if len(hosts) == 0 {
		return Config{}, fmt.Errorf("no hosts configured")
	}
	cfg := Config{
		Hosts:         hosts,
		ProxyMappings: map[string]string{},
		HostAddresses: map[string]string{},
		Defaults:      Defaults{BaseDomain: "local", Scheme: "http"},
	}
	cfg.ProxyMappings = ensureMapping(cfg.ProxyMappings)
	cfg.HostAddresses = ensureMapping(cfg.HostAddresses)
	cfg.Hosts = ensureHostNames(cfg.Hosts)
	cfg.ParsedMappings = parseProxyMappings(cfg.ProxyMappings)
	return cfg, nil
}

func marshalConfigYAML(cfg Config) ([]byte, error) {
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, err
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return data, nil
}

func writeConfigYAML(out io.Writer, cfg Config) error {
	data, err := marshalConfigYAML(cfg)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

func parseHosts(value string) []Host {
	parts := strings.Split(value, ",")
	hosts := make([]Host, 0, len(parts))
	for i, item := range parts {
		endpoint := strings.TrimSpace(item)
		if endpoint == "" {
			continue
		}
		hosts = append(hosts, Host{
			Name:     fmt.Sprintf("host-%d", i+1),
			Endpoint: endpoint,
		})
	}
	return hosts
}

func ensureMapping(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func ensureHostNames(hosts []Host) []Host {
	updated := make([]Host, 0, len(hosts))
	for i, host := range hosts {
		if strings.TrimSpace(host.Name) == "" {
			host.Name = fmt.Sprintf("host-%d", i+1)
		}
		updated = append(updated, host)
	}
	return updated
}

func parseProxyMapping(value string) (ProxyTarget, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ProxyTarget{}, fmt.Errorf("empty mapping value")
	}

	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return ProxyTarget{}, fmt.Errorf("invalid format: expected host/container[:port], got %q", value)
	}

	host := strings.TrimSpace(parts[0])
	containerAndPort := strings.TrimSpace(parts[1])

	if host == "" {
		return ProxyTarget{}, fmt.Errorf("empty host in mapping %q", value)
	}
	if containerAndPort == "" {
		return ProxyTarget{}, fmt.Errorf("empty container in mapping %q", value)
	}

	container := containerAndPort
	port := 0

	if idx := strings.LastIndex(containerAndPort, ":"); idx != -1 {
		container = strings.TrimSpace(containerAndPort[:idx])
		portStr := strings.TrimSpace(containerAndPort[idx+1:])

		if portStr != "" {
			parsed, err := fmt.Sscanf(portStr, "%d", &port)
			if err != nil || parsed != 1 {
				return ProxyTarget{}, fmt.Errorf("invalid port in mapping %q: %q", value, portStr)
			}
			if port <= 0 || port > 65535 {
				return ProxyTarget{}, fmt.Errorf("port out of range in mapping %q: %d", value, port)
			}
		}
	}

	if container == "" {
		return ProxyTarget{}, fmt.Errorf("empty container name in mapping %q", value)
	}

	return ProxyTarget{
		Host:      host,
		Container: container,
		Port:      port,
	}, nil
}

func parseProxyMappings(mappings map[string]string) map[string]ProxyTarget {
	parsed := make(map[string]ProxyTarget)
	for domain, value := range mappings {
		target, err := parseProxyMapping(value)
		if err != nil {
			continue
		}
		parsed[domain] = target
	}
	return parsed
}
