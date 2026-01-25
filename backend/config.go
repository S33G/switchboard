package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

func loadConfigFromEnv() (Config, error) {
	path := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
		cfg.ProxyMappings = ensureMapping(cfg.ProxyMappings)
		cfg.HostAddresses = ensureMapping(cfg.HostAddresses)
		cfg.Hosts = ensureHostNames(cfg.Hosts)
		if len(cfg.Hosts) == 0 {
			return Config{}, fmt.Errorf("no hosts configured")
		}
		return cfg, nil
	}

	return defaultConfigFromEnv()
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
