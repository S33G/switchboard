package main

import (
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
)

type StateStore struct {
	mu         sync.RWMutex
	containers map[string]*Container
}

func NewStateStore() *StateStore {
	return &StateStore{containers: make(map[string]*Container)}
}

func (s *StateStore) Snapshot() []Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Container, 0, len(s.containers))
	for _, container := range s.containers {
		copyItem := *container
		items = append(items, copyItem)
	}
	return items
}

func (s *StateStore) UpdateFromHost(hostName string, containers []container.Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]struct{}, len(containers))
	for _, item := range containers {
		seen[item.ID] = struct{}{}
		name := normalizeName(item.Names)
		labels := item.Labels
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["host"] = hostName
		s.containers[item.ID] = &Container{
			ID:        item.ID,
			Name:      name,
			Image:     item.Image,
			State:     string(item.State),
			Status:    item.Status,
			Host:      hostName,
			Ports:     item.Ports,
			Labels:    labels,
			UpdatedAt: time.Now(),
		}
	}
	for id, existing := range s.containers {
		if existing.Host != hostName {
			continue
		}
		if _, ok := seen[id]; !ok {
			delete(s.containers, id)
		}
	}
}

func normalizeName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}
