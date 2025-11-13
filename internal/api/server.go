package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/company/ems-devices/internal/collectors"
	"github.com/company/ems-devices/internal/config"
)

// Server 提供配置管理和数据源测试的 HTTP API。
type Server struct {
	configPath string
	service    *collectors.Service
	mu         sync.RWMutex
	cfg        *config.Config
}

// NewServer 创建新的 API 服务器。
func NewServer(configPath string, service *collectors.Service) *Server {
	cfg, _ := config.Load(configPath)
	return &Server{
		configPath: configPath,
		service:    service,
		cfg:        cfg,
	}
}

// ServeHTTP 实现 http.Handler 接口。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS 支持
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 记录 API 请求
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/") {
		log.Printf("API 请求: %s %s", r.Method, path)
	}

	// 路由处理
	switch {
	case path == "/api/config" && r.Method == "GET":
		s.handleGetConfig(w, r)
	case path == "/api/config" && r.Method == "PUT":
		s.handleUpdateConfig(w, r)
	case path == "/api/config/validate" && r.Method == "GET":
		s.handleValidateConfig(w, r)
	case path == "/api/config/metrics-url" && r.Method == "GET":
		s.handleGetMetricsURL(w, r)
	case path == "/api/datasource/test/mysql" && r.Method == "POST":
		s.handleTestMySQL(w, r)
	case path == "/api/datasource/test/iotdb" && r.Method == "POST":
		s.handleTestIoTDB(w, r)
	case path == "/api/datasource/test/http_api" && r.Method == "POST":
		s.handleTestHTTPAPI(w, r)
	case path == "/api/datasource/query/preview" && r.Method == "POST":
		s.handlePreviewQuery(w, r)
	case path == "/api/metrics" && r.Method == "GET":
		s.handleListMetrics(w, r)
	case path == "/api/metrics" && r.Method == "POST":
		s.handleCreateMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "GET":
		s.handleGetMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "PUT":
		s.handleUpdateMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "DELETE":
		s.handleDeleteMetric(w, r)
	case path == "/metrics":
		// 转发到 Prometheus handler
		s.service.GetPrometheusHandler().ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}


func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("编码 JSON 响应失败: %v", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) getConfig() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Server) setConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
}
