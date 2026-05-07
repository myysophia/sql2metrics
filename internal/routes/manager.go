package routes

import (
	"log"

	"github.com/company/ems-devices/internal/alerts"
)

// Manager manages route evaluation
type Manager struct {
	storage *RouteStorage
	matcher *Matcher
}

// NewManager creates a new route manager
func NewManager(storage *RouteStorage) *Manager {
	return &Manager{
		storage: storage,
	}
}

// Reload reloads routes from storage and recreates matcher
func (m *Manager) Reload() error {
	if err := m.storage.Load(); err != nil {
		return err
	}

	m.matcher = NewMatcher(m.storage.ListRoutes())
	log.Printf("[ROUTE-MANAGER] Routes reloaded: %d routes, %d channels",
		len(m.storage.ListRoutes()), len(m.storage.ListChannels()))

	return nil
}

// EvaluateRoutes evaluates which channels should receive an alert
func (m *Manager) EvaluateRoutes(alert alerts.Alert, channelIDs []string) []string {
	// Start with direct channel IDs if provided
	resultChannels := make([]string, 0)
	if len(channelIDs) > 0 {
		resultChannels = append(resultChannels, channelIDs...)
	}

	// Evaluate routes if matcher is available
	if m.matcher != nil {
		matchedChannels := m.matcher.Match(alert)
		resultChannels = append(resultChannels, matchedChannels...)
	}

	return resultChannels
}

// GetStorage returns the underlying storage
func (m *Manager) GetStorage() *RouteStorage {
	return m.storage
}
