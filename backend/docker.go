package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type DockerClientManager struct {
	clients map[string]*client.Client
	hosts   []Host
	active  []Host
}

func NewDockerClientManager(hosts []Host) *DockerClientManager {
	return &DockerClientManager{
		clients: make(map[string]*client.Client),
		hosts:   hosts,
		active:  nil,
	}
}

func (m *DockerClientManager) Connect(ctx context.Context) error {
	// Rebuild clients/active list on each connect attempt.
	for _, cli := range m.clients {
		_ = cli.Close()
	}
	m.clients = make(map[string]*client.Client)
	m.active = m.active[:0]

	for _, host := range m.hosts {
		cli, err := m.createClient(ctx, host)
		if err != nil {
			log.Printf("WARN docker connect: create client %s (%s): %v", host.Name, host.Endpoint, err)
			continue
		}
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err = cli.Ping(pingCtx, client.PingOptions{})
		cancel()
		if err != nil {
			_ = cli.Close()
			log.Printf("WARN docker connect: ping %s (%s): %v", host.Name, host.Endpoint, err)
			continue
		}
		m.clients[host.Name] = cli
		m.active = append(m.active, host)
	}

	if len(m.active) == 0 {
		return fmt.Errorf("no docker hosts reachable")
	}
	return nil
}

func (m *DockerClientManager) HostNames() []string {
	names := make([]string, 0, len(m.active))
	for _, host := range m.active {
		names = append(names, host.Name)
	}
	return names
}

func (m *DockerClientManager) ListContainers(ctx context.Context, hostName string) ([]container.Summary, error) {
	cli, ok := m.clients[hostName]
	if !ok {
		return nil, fmt.Errorf("unknown host %s", hostName)
	}
	result, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (m *DockerClientManager) Events(ctx context.Context, hostName string) (client.EventsResult, error) {
	cli, ok := m.clients[hostName]
	if !ok {
		return client.EventsResult{}, fmt.Errorf("unknown host %s", hostName)
	}
	return cli.Events(ctx, client.EventsListOptions{}), nil
}

func (m *DockerClientManager) createClient(ctx context.Context, host Host) (*client.Client, error) {
	endpoint := host.Endpoint
	if name, ok := strings.CutPrefix(endpoint, "context://"); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("empty docker context name")
		}
		resolved, err := resolveDockerContextEndpoint(ctx, name)
		if err != nil {
			return nil, err
		}
		endpoint = resolved
	}

	opts := []client.Opt{client.WithHost(endpoint)}
	if helper, err := connhelper.GetConnectionHelper(endpoint); err == nil && helper != nil {
		httpClient := &http.Client{Transport: &http.Transport{DialContext: helper.Dialer}}
		opts = []client.Opt{client.WithHost(helper.Host), client.WithHTTPClient(httpClient)}
	}
	if host.TLS != nil && host.TLS.CA != "" && host.TLS.Cert != "" && host.TLS.Key != "" {
		opts = append(opts, client.WithTLSClientConfig(host.TLS.CA, host.TLS.Cert, host.TLS.Key))
	}
	cli, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create client %s: %w", host.Name, err)
	}
	return cli, nil
}
