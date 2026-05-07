package routes

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

// RouteStorage manages routes and channels persistence
type RouteStorage struct {
	filepath string
	channels []NotificationChannel
	routes   []AlertRoute
	mu       sync.RWMutex
}

// NewRouteStorage creates a new route storage instance
func NewRouteStorage(filepath string) *RouteStorage {
	return &RouteStorage{
		filepath: filepath,
		channels: make([]NotificationChannel, 0),
		routes:   make([]AlertRoute, 0),
	}
}

// Load loads routes and channels from file
func (s *RouteStorage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filepath); os.IsNotExist(err) {
		// First run, create empty config
		return s.saveLocked()
	}

	data, err := os.ReadFile(s.filepath)
	if err != nil {
		return fmt.Errorf("读取路由配置失败: %w", err)
	}

	var config struct {
		Channels []NotificationChannel `json:"channels"`
		Routes   []AlertRoute          `json:"routes"`
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("解析路由配置失败: %w", err)
		}
	}

	s.channels = config.Channels
	s.routes = config.Routes
	return nil
}

// Save saves routes and channels to file
func (s *RouteStorage) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveLocked()
}

// saveLocked saves without locking (caller must hold lock)
func (s *RouteStorage) saveLocked() error {
	config := struct {
		Channels []NotificationChannel `json:"channels"`
		Routes   []AlertRoute          `json:"routes"`
	}{
		Channels: s.channels,
		Routes:   s.routes,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化路由配置失败: %w", err)
	}

	// Ensure directory exists
	dir := s.filepath[:len(s.filepath)-len("/routes.json")]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(s.filepath, data, 0644); err != nil {
		return fmt.Errorf("写入路由配置失败: %w", err)
	}

	return nil
}

// ==================== Channel CRUD operations ====================

// ListChannels returns all channels
func (s *RouteStorage) ListChannels() []NotificationChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.channels
}

// AddChannel adds a new channel
func (s *RouteStorage) AddChannel(ch NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check duplicate ID
	for _, c := range s.channels {
		if c.ID == ch.ID {
			return fmt.Errorf("渠道 ID %s 已存在", ch.ID)
		}
	}

	s.channels = append(s.channels, ch)
	return s.saveLocked()
}

// UpdateChannel updates an existing channel
func (s *RouteStorage) UpdateChannel(ch NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.channels {
		if c.ID == ch.ID {
			s.channels[i] = ch
			return s.saveLocked()
		}
	}

	return fmt.Errorf("渠道 %s 不存在", ch.ID)
}

// DeleteChannel deletes a channel by ID
func (s *RouteStorage) DeleteChannel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.channels {
		if c.ID == id {
			s.channels = append(s.channels[:i], s.channels[i+1:]...)
			return s.saveLocked()
		}
	}

	return fmt.Errorf("渠道 %s 不存在", id)
}

// GetChannel returns a channel by ID
func (s *RouteStorage) GetChannel(id string) (NotificationChannel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.channels {
		if c.ID == id {
			return c, true
		}
	}
	return NotificationChannel{}, false
}

// ==================== Route CRUD operations ====================

// ListRoutes returns all routes sorted by priority
func (s *RouteStorage) ListRoutes() []AlertRoute {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Copy routes
	routes := make([]AlertRoute, len(s.routes))
	copy(routes, s.routes)

	// Sort by priority (higher first)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Priority > routes[j].Priority
	})

	return routes
}

// AddRoute adds a new route
func (s *RouteStorage) AddRoute(route AlertRoute) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.routes {
		if r.ID == route.ID {
			return fmt.Errorf("路由 ID %s 已存在", route.ID)
		}
	}

	s.routes = append(s.routes, route)
	return s.saveLocked()
}

// UpdateRoute updates an existing route
func (s *RouteStorage) UpdateRoute(route AlertRoute) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.routes {
		if r.ID == route.ID {
			s.routes[i] = route
			return s.saveLocked()
		}
	}

	return fmt.Errorf("路由 %s 不存在", route.ID)
}

// DeleteRoute deletes a route by ID
func (s *RouteStorage) DeleteRoute(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.routes {
		if r.ID == id {
			s.routes = append(s.routes[:i], s.routes[i+1:]...)
			return s.saveLocked()
		}
	}

	return fmt.Errorf("路由 %s 不存在", id)
}

// GetRoute returns a route by ID
func (s *RouteStorage) GetRoute(id string) (AlertRoute, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.routes {
		if r.ID == id {
			return r, true
		}
	}
	return AlertRoute{}, false
}
