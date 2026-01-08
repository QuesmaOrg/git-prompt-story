package session

import (
	"sort"
	"sync"
	"time"
)

var (
	discoverers = make(map[string]SessionDiscoverer)
	parsers     = make(map[string]SessionParser)
	mu          sync.RWMutex
)

// RegisterDiscoverer registers a session discoverer for a prompt tool.
// Should be called during init() of each prompt tool package.
func RegisterDiscoverer(d SessionDiscoverer) {
	mu.Lock()
	defer mu.Unlock()
	discoverers[d.PromptTool()] = d
}

// RegisterParser registers a session parser for a prompt tool.
// Should be called during init() of each prompt tool package.
func RegisterParser(p SessionParser) {
	mu.Lock()
	defer mu.Unlock()
	parsers[p.PromptTool()] = p
}

// GetDiscoverers returns all registered discoverers.
func GetDiscoverers() []SessionDiscoverer {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]SessionDiscoverer, 0, len(discoverers))
	for _, d := range discoverers {
		result = append(result, d)
	}
	return result
}

// GetDiscoverer returns the discoverer for a specific prompt tool, or nil if not found.
func GetDiscoverer(promptTool string) SessionDiscoverer {
	mu.RLock()
	defer mu.RUnlock()
	return discoverers[promptTool]
}

// GetParser returns the parser for a specific prompt tool, or nil if not found.
func GetParser(promptTool string) SessionParser {
	mu.RLock()
	defer mu.RUnlock()
	return parsers[promptTool]
}

// GetParsers returns all registered parsers.
func GetParsers() []SessionParser {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]SessionParser, 0, len(parsers))
	for _, p := range parsers {
		result = append(result, p)
	}
	return result
}

// FindAllSessions discovers sessions from all registered prompt tools.
// Returns sessions sorted by modified time (most recent first).
func FindAllSessions(repoPath string, startWork, endWork time.Time, trace *TraceContext) ([]Session, error) {
	var allSessions []Session

	for _, d := range GetDiscoverers() {
		sessions, err := d.DiscoverSessions(repoPath, startWork, endWork, trace)
		if err != nil {
			// Log but don't fail - continue with other tools
			continue
		}
		allSessions = append(allSessions, sessions...)
	}

	// Sort by modified time (most recent first)
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].GetModified().After(allSessions[j].GetModified())
	})

	return allSessions, nil
}

// CountAllUserActions counts user actions across all sessions using tool-specific logic.
func CountAllUserActions(sessions []Session, startWork, endWork time.Time) int {
	// Group sessions by prompt tool
	byTool := make(map[string][]Session)
	for _, s := range sessions {
		byTool[s.GetPromptTool()] = append(byTool[s.GetPromptTool()], s)
	}

	count := 0
	for toolName, toolSessions := range byTool {
		d := GetDiscoverer(toolName)
		if d != nil {
			count += d.CountUserActions(toolSessions, startWork, endWork)
		}
	}

	return count
}
