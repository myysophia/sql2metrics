package alerts

import (
	"sort"
	"sync"
	"time"
)

// History manages alert history entries
type History struct {
	entries    []AlertHistoryEntry
	maxEntries int
	mu         sync.RWMutex
}

// NewHistory creates a new alert history manager
func NewHistory(maxEntries int) *History {
	if maxEntries <= 0 {
		maxEntries = 1000 // default 1000 entries
	}
	return &History{
		entries:    make([]AlertHistoryEntry, 0),
		maxEntries: maxEntries,
	}
}

// Add adds a new history entry
func (h *History) Add(entry AlertHistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, entry)

	// Keep only maxEntries
	if len(h.entries) > h.maxEntries {
		// Remove oldest entries
		h.entries = h.entries[len(h.entries)-h.maxEntries:]
	}
}

// GetByID returns a history entry by ID
func (h *History) GetByID(id string) (AlertHistoryEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, entry := range h.entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return AlertHistoryEntry{}, false
}

// GetByRuleID returns history entries for a specific rule
func (h *History) GetByRuleID(ruleID string, limit int) []AlertHistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []AlertHistoryEntry
	for _, entry := range h.entries {
		if entry.AlertRuleID == ruleID {
			result = append(result, entry)
		}
	}

	// Sort by triggered_at descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, result[i].TriggeredAt)
		timeJ, _ := time.Parse(time.RFC3339, result[j].TriggeredAt)
		return timeI.After(timeJ)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result
}

// List returns all history entries with pagination
func (h *History) List(page, pageSize int) ([]AlertHistoryEntry, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := len(h.entries)

	// Sort by triggered_at descending (newest first)
	sorted := make([]AlertHistoryEntry, len(h.entries))
	copy(sorted, h.entries)
	sort.Slice(sorted, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, sorted[i].TriggeredAt)
		timeJ, _ := time.Parse(time.RFC3339, sorted[j].TriggeredAt)
		return timeI.After(timeJ)
	})

	// Pagination
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > len(sorted) {
		end = len(sorted)
	}

	if start >= len(sorted) {
		return []AlertHistoryEntry{}, total
	}

	return sorted[start:end], total
}

// GetActiveFiring returns all currently firing alerts
func (h *History) GetActiveFiring() []AlertHistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []AlertHistoryEntry
	for _, entry := range h.entries {
		if entry.State == "firing" && entry.ResolvedAt == "" {
			result = append(result, entry)
		}
	}
	return result
}

// UpdateResolved updates the resolved timestamp for a firing alert
func (h *History) UpdateResolved(ruleID string, resolvedAt time.Time) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find the firing entry for this rule
	for i := len(h.entries) - 1; i >= 0; i-- {
		if h.entries[i].AlertRuleID == ruleID &&
			h.entries[i].State == "firing" &&
			h.entries[i].ResolvedAt == "" {
			h.entries[i].ResolvedAt = resolvedAt.Format(time.RFC3339)
			return true
		}
	}

	return false
}

// GetFiringEntryForRule returns the current firing entry for a rule
func (h *History) GetFiringEntryForRule(ruleID string) (AlertHistoryEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Search from newest to oldest
	for i := len(h.entries) - 1; i >= 0; i-- {
		if h.entries[i].AlertRuleID == ruleID &&
			h.entries[i].State == "firing" &&
			h.entries[i].ResolvedAt == "" {
			return h.entries[i], true
		}
	}

	return AlertHistoryEntry{}, false
}

// GetStats returns statistics about alert history
func (h *History) GetStats() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]int{
		"total":    len(h.entries),
		"firing":   0,
		"resolved": 0,
	}

	for _, entry := range h.entries {
		if entry.State == "firing" {
			if entry.ResolvedAt == "" {
				stats["firing"]++
			} else {
				stats["resolved"]++
			}
		}
	}

	return stats
}

// Clear removes all history entries
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = make([]AlertHistoryEntry, 0)
}

// GetCount returns the number of entries
func (h *History) GetCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.entries)
}
