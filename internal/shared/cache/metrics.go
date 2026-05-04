package cache

import (
	"sort"
	"sync"
	"sync/atomic"
)

type NamespaceMetrics struct {
	Hits   uint64 `json:"hits"`
	Misses uint64 `json:"misses"`
	Errors uint64 `json:"errors"`
	Sets   uint64 `json:"sets"`
}

type counters struct {
	hits   atomic.Uint64
	misses atomic.Uint64
	errors atomic.Uint64
	sets   atomic.Uint64
}

var (
	metricsMu sync.RWMutex
	metrics   = make(map[string]*counters)
)

func entryFor(namespace string) *counters {
	metricsMu.RLock()
	c, ok := metrics[namespace]
	metricsMu.RUnlock()
	if ok {
		return c
	}

	metricsMu.Lock()
	defer metricsMu.Unlock()
	if existing, ok := metrics[namespace]; ok {
		return existing
	}
	c = &counters{}
	metrics[namespace] = c
	return c
}

func RecordHit(namespace string) {
	entryFor(namespace).hits.Add(1)
}

func RecordMiss(namespace string) {
	entryFor(namespace).misses.Add(1)
}

func RecordError(namespace string) {
	entryFor(namespace).errors.Add(1)
}

func RecordSet(namespace string) {
	entryFor(namespace).sets.Add(1)
}

func Snapshot() map[string]NamespaceMetrics {
	metricsMu.RLock()
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	metricsMu.RUnlock()

	sort.Strings(keys)

	out := make(map[string]NamespaceMetrics, len(keys))
	for _, key := range keys {
		c := entryFor(key)
		out[key] = NamespaceMetrics{
			Hits:   c.hits.Load(),
			Misses: c.misses.Load(),
			Errors: c.errors.Load(),
			Sets:   c.sets.Load(),
		}
	}
	return out
}

func Reset() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	metrics = make(map[string]*counters)
}
