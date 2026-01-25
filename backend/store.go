package main

import (
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
)

type ContainerDiff struct {
	Added   []Container `json:"added"`
	Updated []Container `json:"updated"`
	Removed []string    `json:"removed"`
}

func NewContainerDiff() ContainerDiff {
	return ContainerDiff{
		Added:   []Container{},
		Updated: []Container{},
		Removed: []string{},
	}
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
		log.Printf("DIFF SENT: added=%d updated=%d removed=%d", len(diff.Added), len(diff.Updated), len(diff.Removed))
	default:
		log.Printf("DIFF DROPPED (channel full): added=%d updated=%d removed=%d", len(diff.Added), len(diff.Updated), len(diff.Removed))
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

func (s *StateStore) UpdateSingleContainer(hostName string, item dockercontainer.Summary) {
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
		Ports:     convertPorts(item.Ports, hostName, name),
		Labels:    labels,
		UpdatedAt: time.Now(),
	}

	oldMap := s.containers.Load().(map[string]*Container)
	newMap := make(map[string]*Container, len(oldMap)+1)
	for k, v := range oldMap {
		newMap[k] = v
	}

	diff := NewContainerDiff()
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

func (s *StateStore) UpdateFromHost(hostName string, containers []dockercontainer.Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldMap := s.containers.Load().(map[string]*Container)
	newMap := make(map[string]*Container)

	diff := NewContainerDiff()
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
			Ports:     convertPorts(item.Ports, hostName, name),
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

func convertPorts(dockerPorts []dockercontainer.Port, hostName string, containerName string) []Port {
	ports := make([]Port, len(dockerPorts))
	for i, dp := range dockerPorts {
		p := Port{
			Private: uint16(dp.PrivatePort),
			Public:  dp.PublicPort,
			Type:    dp.Type,
		}
		if p.Public > 0 {
			p.Proxied = isPortProxied(hostName, containerName, int(p.Public))
		}
		ports[i] = p
	}

	result := make([]Port, len(ports))
	copy(result, ports)

	result2 := Container{Ports: result}
	result2.SortPorts()

	return result2.Ports
}

var proxiedPortsInfo = struct {
	mu    sync.RWMutex
	ports map[string]map[string]bool
}{
	ports: make(map[string]map[string]bool),
}

func setProxiedPorts(ports map[string]map[string][]string) {
	proxiedPortsInfo.mu.Lock()
	defer proxiedPortsInfo.mu.Unlock()

	newPorts := make(map[string]map[string]bool)
	for targetStr, targetMap := range ports {
		for target := range targetMap {
			key := normalizeProxyKey(targetStr)
			if newPorts[key] == nil {
				newPorts[key] = make(map[string]bool)
			}
			newPorts[key][target] = true
		}
	}
	proxiedPortsInfo.ports = newPorts
}

func normalizeProxyKey(target string) string {
	parts := strings.Split(target, "/")
	if len(parts) >= 2 {
		containerPart := parts[1]
		colonIdx := strings.LastIndex(containerPart, ":")
		if colonIdx != -1 {
			return parts[0] + "/" + containerPart[:colonIdx]
		}
		return parts[0] + "/" + containerPart
	}
	return target
}

func isPortProxied(hostName string, containerName string, port int) bool {
	proxiedPortsInfo.mu.RLock()
	defer proxiedPortsInfo.mu.RUnlock()

	key := hostName + "/" + containerName
	if proxiedTargets, ok := proxiedPortsInfo.ports[key]; ok {
		return len(proxiedTargets) > 0
	}
	return false
}
