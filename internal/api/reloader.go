package api

import (
	"fmt"
	"sync"

	"github.com/company/ems-devices/internal/collectors"
	"github.com/company/ems-devices/internal/config"
)

// ReloadResult 表示热更新结果。
type ReloadResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error,omitempty"`
}

// Reloader 负责管理配置热更新。
type Reloader struct {
	service    *collectors.Service
	mu         sync.RWMutex
	configPath string
}

// NewReloader 创建新的热更新器。
func NewReloader(service *collectors.Service, configPath string) *Reloader {
	return &Reloader{
		service:    service,
		configPath: configPath,
	}
}

// Reload 重新加载配置并更新服务。
func (r *Reloader) Reload(cfg *config.Config) ReloadResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 创建新的服务实例
	newService, err := collectors.NewService(cfg)
	if err != nil {
		return ReloadResult{
			Success: false,
			Error:   fmt.Errorf("创建新服务失败: %w", err),
		}
	}

	// 关闭旧服务
	if r.service != nil {
		r.service.Close()
	}

	// 更新服务引用
	r.service = newService

	return ReloadResult{
		Success: true,
		Message: "配置热更新成功",
	}
}

// GetService 获取当前服务实例。
func (r *Reloader) GetService() *collectors.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.service
}
