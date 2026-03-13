package alerts

import (
	"sort"
	"sync"
	"time"
)

// MetricValuePoint represents a metric value at a specific time
type MetricValuePoint struct {
	Timestamp time.Time
	Value     float64
}

// MetricValueStore stores historical metric values for trend and anomaly analysis
type MetricValueStore struct {
	values map[string][]MetricValuePoint
	mu     sync.RWMutex
	maxAge time.Duration // Keep data for maxAge (default 48h)
}

// NewMetricValueStore creates a new metric value store
func NewMetricValueStore(maxAge time.Duration) *MetricValueStore {
	if maxAge <= 0 {
		maxAge = 48 * time.Hour // default 48 hours
	}
	return &MetricValueStore{
		values: make(map[string][]MetricValuePoint),
		maxAge: maxAge,
	}
}

// AddValue adds a new metric value
func (s *MetricValueStore) AddValue(metricName string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	point := MetricValuePoint{
		Timestamp: time.Now(),
		Value:     value,
	}

	s.values[metricName] = append(s.values[metricName], point)

	// Cleanup old data periodically
	s.cleanupUnsafe(metricName)
}

// GetValues retrieves values within a time window
func (s *MetricValueStore) GetValues(metricName string, window time.Duration) []MetricValuePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, ok := s.values[metricName]
	if !ok {
		return []MetricValuePoint{}
	}

	// Filter by time window
	cutoff := time.Now().Add(-window)
	var result []MetricValuePoint
	for _, p := range points {
		if p.Timestamp.After(cutoff) {
			result = append(result, p)
		}
	}

	// Sort by timestamp
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetLatestValue returns the most recent value for a metric
func (s *MetricValueStore) GetLatestValue(metricName string) (MetricValuePoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, ok := s.values[metricName]
	if !ok || len(points) == 0 {
		return MetricValuePoint{}, false
	}

	return points[len(points)-1], true
}

// GetValuesSince retrieves values since a specific time
func (s *MetricValueStore) GetValuesSince(metricName string, since time.Time) []MetricValuePoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	points, ok := s.values[metricName]
	if !ok {
		return []MetricValuePoint{}
	}

	var result []MetricValuePoint
	for _, p := range points {
		if p.Timestamp.After(since) {
			result = append(result, p)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

// GetMetricNames returns all metric names in the store
func (s *MetricValueStore) GetMetricNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.values))
	for name := range s.values {
		names = append(names, name)
	}
	return names
}

// RemoveMetric removes all values for a specific metric
func (s *MetricValueStore) RemoveMetric(metricName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, metricName)
}

// Clear removes all stored values
func (s *MetricValueStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values = make(map[string][]MetricValuePoint)
}

// cleanupUnsafe removes old data points (must be called with lock held)
func (s *MetricValueStore) cleanupUnsafe(metricName string) {
	points := s.values[metricName]
	cutoff := time.Now().Add(-s.maxAge)

	// Find first valid index
	firstValid := 0
	for i, p := range points {
		if p.Timestamp.After(cutoff) {
			firstValid = i
			break
		}
	}

	// Keep only recent data
	if firstValid > 0 {
		s.values[metricName] = points[firstValid:]
	}
}

// Cleanup removes old data points for all metrics
func (s *MetricValueStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for metricName := range s.values {
		s.cleanupUnsafe(metricName)
	}
}

// GetCount returns the number of data points for a metric
func (s *MetricValueStore) GetCount(metricName string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.values[metricName])
}

// GetSize returns the total number of data points across all metrics
func (s *MetricValueStore) GetSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	for _, points := range s.values {
		total += len(points)
	}
	return total
}
