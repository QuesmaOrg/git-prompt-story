package provider

import (
	"sync"
)

var (
	providers = make(map[string]Provider)
	mu        sync.RWMutex
)

// Register adds a provider to the registry
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	providers[p.Name()] = p
}

// Get returns a provider by name
func Get(name string) Provider {
	mu.RLock()
	defer mu.RUnlock()
	return providers[name]
}

// All returns all registered providers
func All() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Provider, 0, len(providers))
	for _, p := range providers {
		result = append(result, p)
	}
	return result
}

// Names returns the names of all registered providers
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}
