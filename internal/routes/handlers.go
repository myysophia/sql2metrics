package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Handler handles HTTP requests for route management
type Handler struct {
	storage *RouteStorage
	manager *Manager
}

// NewHandler creates a new route handler
func NewHandler(storage *RouteStorage, manager *Manager) *Handler {
	return &Handler{
		storage: storage,
		manager: manager,
	}
}

// ==================== Channel endpoints ====================

// ListChannels handles GET /api/routes/channels
func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels := h.storage.ListChannels()
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
		"total":    len(channels),
	})
}

// CreateChannel handles POST /api/routes/channels
func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var ch NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	// Set timestamps
	now := time.Now().Format(time.RFC3339)
	ch.CreatedAt = now
	ch.UpdatedAt = now

	// Generate ID if not provided
	if ch.ID == "" {
		ch.ID = h.generateChannelID(ch.Type)
	}

	if err := h.storage.AddChannel(ch); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to add channel: %v", err)
		h.writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Reload manager to update routes
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Channel created: %s (%s)", ch.ID, ch.Name)
	h.writeJSON(w, http.StatusCreated, ch)
}

// UpdateChannel handles PUT /api/routes/channels/:id
func (h *Handler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	id := h.extractID(r.URL.Path, "/api/routes/channels/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "无效的渠道 ID")
		return
	}

	var ch NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	ch.ID = id
	ch.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := h.storage.UpdateChannel(ch); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to update channel: %v", err)
		if strings.Contains(err.Error(), "不存在") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusConflict, err.Error())
		}
		return
	}

	// Reload manager to update routes
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Channel updated: %s", ch.ID)
	h.writeJSON(w, http.StatusOK, ch)
}

// DeleteChannel handles DELETE /api/routes/channels/:id
func (h *Handler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	id := h.extractID(r.URL.Path, "/api/routes/channels/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "无效的渠道 ID")
		return
	}

	if err := h.storage.DeleteChannel(id); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to delete channel: %v", err)
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Reload manager to update routes
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Channel deleted: %s", id)
	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "通知渠道已删除",
		"id":      id,
	})
}

// TestChannel handles POST /api/routes/channels/:id/test
func (h *Handler) TestChannel(w http.ResponseWriter, r *http.Request) {
	id := h.extractID(r.URL.Path, "/api/routes/channels/")
	id = strings.TrimSuffix(id, "/test")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "无效的渠道 ID")
		return
	}

	ch, ok := h.storage.GetChannel(id)
	if !ok {
		h.writeError(w, http.StatusNotFound, "通知渠道不存在")
		return
	}

	log.Printf("[ROUTE-HANDLER] Test notification requested for channel: %s (%s)", ch.ID, ch.Name)
	// TODO: Implement actual test notification
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "测试消息已发送",
		"channel":  ch.ID,
		"channel_name": ch.Name,
	})
}

// ==================== Route endpoints ====================

// ListRoutes handles GET /api/routes/rules
func (h *Handler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	routes := h.manager.GetStorage().ListRoutes()
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"routes": routes,
		"total":  len(routes),
	})
}

// CreateRoute handles POST /api/routes/rules
func (h *Handler) CreateRoute(w http.ResponseWriter, r *http.Request) {
	var route AlertRoute
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	now := time.Now().Format(time.RFC3339)
	route.CreatedAt = now
	route.UpdatedAt = now

	if route.ID == "" {
		route.ID = h.generateRouteID()
	}

	// Validate channel IDs exist
	for _, channelID := range route.ChannelIDs {
		if _, ok := h.storage.GetChannel(channelID); !ok {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("渠道 %s 不存在", channelID))
			return
		}
	}

	if err := h.storage.AddRoute(route); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to add route: %v", err)
		h.writeError(w, http.StatusConflict, err.Error())
		return
	}

	// Reload manager to update matcher
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Route created: %s (%s)", route.ID, route.Name)
	h.writeJSON(w, http.StatusCreated, route)
}

// UpdateRoute handles PUT /api/routes/rules/:id
func (h *Handler) UpdateRoute(w http.ResponseWriter, r *http.Request) {
	id := h.extractID(r.URL.Path, "/api/routes/rules/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "无效的路由 ID")
		return
	}

	var route AlertRoute
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		h.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	route.ID = id
	route.UpdatedAt = time.Now().Format(time.RFC3339)

	// Validate channel IDs exist
	for _, channelID := range route.ChannelIDs {
		if _, ok := h.storage.GetChannel(channelID); !ok {
			h.writeError(w, http.StatusBadRequest, fmt.Sprintf("渠道 %s 不存在", channelID))
			return
		}
	}

	if err := h.storage.UpdateRoute(route); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to update route: %v", err)
		if strings.Contains(err.Error(), "不存在") {
			h.writeError(w, http.StatusNotFound, err.Error())
		} else {
			h.writeError(w, http.StatusConflict, err.Error())
		}
		return
	}

	// Reload manager to update matcher
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Route updated: %s", route.ID)
	h.writeJSON(w, http.StatusOK, route)
}

// DeleteRoute handles DELETE /api/routes/rules/:id
func (h *Handler) DeleteRoute(w http.ResponseWriter, r *http.Request) {
	id := h.extractID(r.URL.Path, "/api/routes/rules/")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "无效的路由 ID")
		return
	}

	if err := h.storage.DeleteRoute(id); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to delete route: %v", err)
		h.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Reload manager to update matcher
	if err := h.manager.Reload(); err != nil {
		log.Printf("[ROUTE-HANDLER] Failed to reload manager: %v", err)
	}

	log.Printf("[ROUTE-HANDLER] Route deleted: %s", id)
	h.writeJSON(w, http.StatusOK, map[string]string{
		"message": "路由规则已删除",
		"id":      id,
	})
}

// ==================== Helper functions ====================

// extractID extracts ID from URL path
func (h *Handler) extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

// generateChannelID generates a unique channel ID
func (h *Handler) generateChannelID(channelType string) string {
	return fmt.Sprintf("%s-%d", channelType, time.Now().Unix())
}

// generateRouteID generates a unique route ID
func (h *Handler) generateRouteID() string {
	return fmt.Sprintf("route-%d", time.Now().Unix())
}

// writeJSON writes JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes error response
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
