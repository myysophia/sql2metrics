package routes

import (
	"regexp"
	"strings"

	"github.com/company/ems-devices/internal/alerts"
)

// Matcher evaluates if an alert matches routes
type Matcher struct {
	routes []AlertRoute
}

// NewMatcher creates a new matcher with routes
func NewMatcher(routes []AlertRoute) *Matcher {
	// Sort routes by priority (higher first)
	sortedRoutes := make([]AlertRoute, len(routes))
	copy(sortedRoutes, routes)

	// Simple sort by priority descending
	for i := 0; i < len(sortedRoutes)-1; i++ {
		for j := 0; j < len(sortedRoutes)-i-1; j++ {
			if sortedRoutes[j].Priority < sortedRoutes[j+1].Priority {
				sortedRoutes[j], sortedRoutes[j+1] = sortedRoutes[j+1], sortedRoutes[j]
			}
		}
	}

	return &Matcher{routes: sortedRoutes}
}

// Match returns all matching channel IDs for an alert
func (m *Matcher) Match(alert alerts.Alert) []string {
	channelIDs := make([]string, 0)

	for _, route := range m.routes {
		if !route.Enabled {
			continue
		}

		if m.matchRoute(route, alert) {
			channelIDs = append(channelIDs, route.ChannelIDs...)

			if !route.Continue {
				break // Stop if route is terminal
			}
		}
	}

	return uniqueIDs(channelIDs)
}

// matchRoute checks if an alert matches a route
func (m *Matcher) matchRoute(route AlertRoute, alert alerts.Alert) bool {
	match := route.Match

	// Check labels (exact match)
	for k, v := range match.Labels {
		alertVal, ok := alert.Labels[k]
		if !ok || alertVal != v {
			return false
		}
	}

	// Check labels (regex match)
	for k, pattern := range match.LabelRegex {
		alertVal, ok := alert.Labels[k]
		if !ok {
			return false
		}

		matched, err := regexp.MatchString(pattern, alertVal)
		if err != nil || !matched {
			return false
		}
	}

	// Check severity
	if len(match.Severities) > 0 {
		severity := alert.Labels["severity"]
		found := false
		for _, s := range match.Severities {
			if strings.EqualFold(s, severity) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check alert names (if specified as comma-separated list)
	if match.AlertNames != "" {
		alertNames := strings.Split(match.AlertNames, ",")
		found := false
		for _, name := range alertNames {
			if strings.TrimSpace(name) == alert.RuleName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check alert name regex
	if match.AlertNameRegex != "" {
		matched, err := regexp.MatchString(match.AlertNameRegex, alert.RuleName)
		if err != nil || !matched {
			return false
		}
	}

	// Check metric names (if specified as comma-separated list)
	if match.MetricNames != "" {
		metricNames := strings.Split(match.MetricNames, ",")
		metricName := alert.Labels["metric_name"]
		found := false
		for _, name := range metricNames {
			if strings.TrimSpace(name) == metricName {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check metric name regex
	if match.MetricNameRegex != "" {
		metricName := alert.Labels["metric_name"]
		matched, err := regexp.MatchString(match.MetricNameRegex, metricName)
		if err != nil || !matched {
			return false
		}
	}

	return true
}

// uniqueIDs removes duplicate IDs from a slice
func uniqueIDs(ids []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}
