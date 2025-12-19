package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/company/ems-devices/internal/collectors"
	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/web"
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
	case path == "/api/datasource/test/redis" && r.Method == "POST":
		s.handleTestRedis(w, r)
	case path == "/api/datasource/test/restapi" && r.Method == "POST":
		s.handleTestRestAPI(w, r)
	case path == "/api/datasource/restapi/preview" && r.Method == "POST":
		s.handlePreviewRestAPI(w, r)
	case path == "/api/datasource/query/preview" && r.Method == "POST":
		s.handlePreviewQuery(w, r)
	case path == "/api/metrics" && r.Method == "GET":
		s.handleListMetrics(w, r)
	case path == "/api/metrics" && r.Method == "POST":
		s.handleCreateMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/index/") && r.Method == "DELETE":
		s.handleDeleteMetricByIndex(w, r)
	case strings.HasPrefix(path, "/api/metrics/index/") && r.Method == "PUT":
		s.handleUpdateMetricByIndex(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "GET":
		s.handleGetMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "PUT":
		s.handleUpdateMetric(w, r)
	case strings.HasPrefix(path, "/api/metrics/") && r.Method == "DELETE":
		s.handleDeleteMetric(w, r)

	// 独立数据源 API
	case strings.HasPrefix(path, "/api/datasource/mysql/") && r.Method == "PUT":
		name := strings.TrimPrefix(path, "/api/datasource/mysql/")
		s.handleUpdateMySQLConnection(w, r, name)
	case strings.HasPrefix(path, "/api/datasource/mysql/") && r.Method == "DELETE":
		name := strings.TrimPrefix(path, "/api/datasource/mysql/")
		s.handleDeleteMySQLConnection(w, r, name)
	case strings.HasPrefix(path, "/api/datasource/redis/") && r.Method == "PUT":
		name := strings.TrimPrefix(path, "/api/datasource/redis/")
		s.handleUpdateRedisConnection(w, r, name)
	case strings.HasPrefix(path, "/api/datasource/redis/") && r.Method == "DELETE":
		name := strings.TrimPrefix(path, "/api/datasource/redis/")
		s.handleDeleteRedisConnection(w, r, name)
	case strings.HasPrefix(path, "/api/datasource/restapi/") && !strings.HasSuffix(path, "/preview") && r.Method == "PUT":
		name := strings.TrimPrefix(path, "/api/datasource/restapi/")
		s.handleUpdateRestAPIConnection(w, r, name)
	case strings.HasPrefix(path, "/api/datasource/restapi/") && !strings.HasSuffix(path, "/preview") && r.Method == "DELETE":
		name := strings.TrimPrefix(path, "/api/datasource/restapi/")
		s.handleDeleteRestAPIConnection(w, r, name)
	case path == "/api/datasource/iotdb" && r.Method == "PUT":
		s.handleUpdateIoTDB(w, r)

	// 独立指标 API (新增)
	case path == "/api/metrics/add" && r.Method == "POST":
		s.handleAddMetric(w, r)

	case path == "/metrics":
		s.service.GetPrometheusHandler().ServeHTTP(w, r)
	default:
		// 尝试从嵌入的静态文件中服务
		distFS, err := web.GetDistFS()
		if err != nil {
			log.Printf("获取静态文件系统失败: %v", err)
			http.NotFound(w, r)
			return
		}

		// 检查文件是否存在
		f, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			defer f.Close()
			http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
			return
		}

		// 如果不是 API 请求且文件不存在，返回 index.html (SPA 支持)
		// 但要排除带扩展名的静态资源请求 (如 .js, .css, .png 等)
		if !strings.Contains(path, ".") {
			indexFile, err := distFS.Open("index.html")
			if err != nil {
				log.Printf("无法打开 index.html: %v", err)
				http.NotFound(w, r)
				return
			}
			defer indexFile.Close()
			
			// 读取 index.html 内容并写入响应
			stat, _ := indexFile.Stat()
			http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
			return
		}

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
