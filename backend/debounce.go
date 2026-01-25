package main

import (
	"sync"
	"time"
)

type debounceTracker struct {
	mu            sync.Mutex
	recentChanges []time.Time
}

func newDebounceTracker() *debounceTracker {
	return &debounceTracker{
		recentChanges: make([]time.Time, 0, 50),
	}
}

func (d *debounceTracker) recordChange() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	d.recentChanges = append(d.recentChanges, now)

	cutoff := now.Add(-10 * time.Second)
	filtered := d.recentChanges[:0]
	for _, t := range d.recentChanges {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	d.recentChanges = filtered
}

func (d *debounceTracker) calculateDebounce(baseDebounce time.Duration) time.Duration {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Second)
	count := 0
	for _, t := range d.recentChanges {
		if t.After(cutoff) {
			count++
		}
	}

	switch {
	case count > 20:
		return 5 * time.Second
	case count > 10:
		return 2 * time.Second
	case count > 5:
		return baseDebounce
	default:
		if baseDebounce > 500*time.Millisecond {
			return 500 * time.Millisecond
		}
		return baseDebounce
	}
}
