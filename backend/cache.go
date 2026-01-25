package main

import (
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

type containerCache struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	container container.Summary
	expiresAt time.Time
}

func newContainerCache() *containerCache {
	return &containerCache{
		cache: make(map[string]*cacheEntry),
	}
}

func (c *containerCache) Get(id string) (*container.Summary, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[id]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return &entry.container, true
}

func (c *containerCache) Set(id string, summary container.Summary, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[id] = &cacheEntry{
		container: summary,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *containerCache) Invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, id)
}

func (c *containerCache) InvalidateHost(hostName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, entry := range c.cache {
		if len(entry.container.Labels) > 0 && entry.container.Labels["host"] == hostName {
			delete(c.cache, id)
		}
	}
}

func (c *containerCache) Clean() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for id, entry := range c.cache {
		if now.After(entry.expiresAt) {
			delete(c.cache, id)
		}
	}
}
