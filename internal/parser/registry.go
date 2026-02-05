package parser

import (
	"sync"
)

var (
	parsers = make(map[string]Parser)
	mu      sync.RWMutex
)

// Register adds a parser to the registry
func Register(p Parser) {
	mu.Lock()
	defer mu.Unlock()
	parsers[p.Name()] = p
}

// Get returns a parser by name
func Get(name string) Parser {
	mu.RLock()
	defer mu.RUnlock()
	return parsers[name]
}

// All returns all registered parsers
func All() []Parser {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Parser, 0, len(parsers))
	for _, p := range parsers {
		result = append(result, p)
	}
	return result
}
