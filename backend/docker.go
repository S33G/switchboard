package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type DockerClientManager struct {
	clients map[string]*client.Client
	hosts   []Host
	active  []Host
	cache   *containerCache
}

func NewDockerClientManager(hosts []Host) *DockerClientManager {
	return &DockerClientManager{
		clients: make(map[string]*client.Client),
		hosts:   hosts,
		active:  nil,
		cache:   newContainerCache(),
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
		_, err = cli.Ping(pingCtx)
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
	result, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (m *DockerClientManager) InspectContainer(ctx context.Context, hostName string, containerID string) (*container.Summary, error) {
	if cached, ok := m.cache.Get(containerID); ok {
		return cached, nil
	}

	cli, ok := m.clients[hostName]
	if !ok {
		return nil, fmt.Errorf("unknown host %s", hostName)
	}

	f := filters.NewArgs()
	f.Add("id", containerID)

	result, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("container %s not found", containerID)
	}

	m.cache.Set(containerID, result[0], 30*time.Second)
	return &result[0], nil
}

type EventsResult struct {
	Messages <-chan events.Message
	Err      <-chan error
}

func (m *DockerClientManager) Events(ctx context.Context, hostName string) (EventsResult, error) {
	cli, ok := m.clients[hostName]
	if !ok {
		return EventsResult{}, fmt.Errorf("unknown host %s", hostName)
	}
	messages, errs := cli.Events(ctx, events.ListOptions{})
	return EventsResult{Messages: messages, Err: errs}, nil
}

func (m *DockerClientManager) Ping(ctx context.Context, hostName string) error {
	cli, ok := m.clients[hostName]
	if !ok {
		return fmt.Errorf("unknown host %s", hostName)
	}
	_, err := cli.Ping(ctx)
	return err
}

func (m *DockerClientManager) ReconnectHost(ctx context.Context, hostName string) error {
	var targetHost *Host
	for i := range m.hosts {
		if m.hosts[i].Name == hostName {
			targetHost = &m.hosts[i]
			break
		}
	}
	if targetHost == nil {
		return fmt.Errorf("host %s not found in configuration", hostName)
	}

	if oldCli, ok := m.clients[hostName]; ok {
		_ = oldCli.Close()
		delete(m.clients, hostName)
	}

	cli, err := m.createClient(ctx, *targetHost)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := cli.Ping(pingCtx); err != nil {
		_ = cli.Close()
		return fmt.Errorf("ping: %w", err)
	}

	m.clients[hostName] = cli
	return nil
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
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("create client %s: %w", host.Name, err)
	}
	return cli, nil
}
