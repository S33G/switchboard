package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type dockerContextInspect struct {
	Endpoints map[string]struct {
		Host string `json:"Host"`
	} `json:"Endpoints"`
}

func resolveDockerContextEndpoint(ctx context.Context, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty docker context name")
	}

	out, err := exec.CommandContext(ctx, "docker", "context", "inspect", name, "--format", "{{json .}}").Output()
	if err != nil {
		return "", fmt.Errorf("docker context inspect %s: %w", name, err)
	}

	var inspected dockerContextInspect
	if err := json.Unmarshal(out, &inspected); err != nil {
		return "", fmt.Errorf("parse docker context inspect %s: %w", name, err)
	}

	if inspected.Endpoints == nil {
		return "", fmt.Errorf("docker context %s has no endpoints", name)
	}
	e, ok := inspected.Endpoints["docker"]
	if !ok {
		return "", fmt.Errorf("docker context %s has no docker endpoint", name)
	}
	endpoint := strings.TrimSpace(e.Host)
	if endpoint == "" {
		return "", fmt.Errorf("docker context %s has no docker endpoint", name)
	}
	return endpoint, nil
}
