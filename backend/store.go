package main

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/container"
)

type ContainerDiff struct {
	Added   []Container `json:"added"`
	Updated []Container `json:"updated"`
	Removed []string    `json:"removed"`
}

type StateStore struct {
	mu         sync.Mutex
	containers atomic.Value
	onChange   chan struct{}
	onDiff     chan ContainerDiff
}

func NewStateStore() *StateStore {
	s := &StateStore{
		onChange: make(chan struct{}, 1),
		onDiff:   make(chan ContainerDiff, 1),
	}
	s.containers.Store(make(map[string]*Container))
	return s
}

func (s *StateStore) Changed() <-chan struct{} {
	return s.onChange
}

func (s *StateStore) Diffs() <-chan ContainerDiff {
	return s.onDiff
}

func (s *StateStore) notifyChange() {
	select {
	case s.onChange <- struct{}{}:
	default:
	}
}

func (s *StateStore) notifyDiff(diff ContainerDiff) {
	select {
	case s.onDiff <- diff:
	default:
	}
}

func (s *StateStore) Snapshot() []Container {
	containerMap := s.containers.Load().(map[string]*Container)
	items := make([]Container, 0, len(containerMap))
	for _, container := range containerMap {
		copyItem := *container
		items = append(items, copyItem)
	}
	return items
}

func (s *StateStore) UpdateSingleContainer(hostName string, item container.Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := normalizeName(item.Names)
	labels := item.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["host"] = hostName

	newContainer := Container{
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

	oldMap := s.containers.Load().(map[string]*Container)
	newMap := make(map[string]*Container, len(oldMap)+1)
	for k, v := range oldMap {
		newMap[k] = v
	}

	var diff ContainerDiff
	if _, exists := oldMap[item.ID]; exists {
		diff.Updated = []Container{newContainer}
	} else {
		diff.Added = []Container{newContainer}
	}

	newMap[item.ID] = &newContainer
	s.containers.Store(newMap)
	s.notifyChange()
	s.notifyDiff(diff)
}

func (s *StateStore) UpdateFromHost(hostName string, containers []container.Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldMap := s.containers.Load().(map[string]*Container)
	newMap := make(map[string]*Container)

	var diff ContainerDiff
	seen := make(map[string]struct{}, len(containers))

	for _, item := range containers {
		seen[item.ID] = struct{}{}
		name := normalizeName(item.Names)
		labels := item.Labels
		if labels == nil {
			labels = make(map[string]string)
		}
		labels["host"] = hostName

		newContainer := Container{
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

		if _, exists := oldMap[item.ID]; exists {
			diff.Updated = append(diff.Updated, newContainer)
		} else {
			diff.Added = append(diff.Added, newContainer)
		}

		newMap[item.ID] = &newContainer
	}

	for id, existing := range oldMap {
		if existing.Host != hostName {
			newMap[id] = existing
			continue
		}
		if _, ok := seen[id]; !ok {
			diff.Removed = append(diff.Removed, id)
		}
	}

	s.containers.Store(newMap)
	s.notifyChange()
	if len(diff.Added) > 0 || len(diff.Updated) > 0 || len(diff.Removed) > 0 {
		s.notifyDiff(diff)
	}
}

func normalizeName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}
