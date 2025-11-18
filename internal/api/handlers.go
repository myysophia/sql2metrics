package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/datasource"
)

// handleGetConfig 获取当前配置。
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	s.writeJSON(w, http.StatusOK, cfg)
}

// handleUpdateConfig 更新配置并触发热更新。
func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	// 应用默认值
	if err := newCfg.ApplyDefaults(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("应用默认值失败: %v", err))
		return
	}

	// 验证配置
	if err := newCfg.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	// 保存到文件
	if err := newCfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	// 触发热更新
	reloadResult := s.service.ReloadConfig(&newCfg)
	if !reloadResult.Success {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("热更新失败: %s", reloadResult.Error))
		return
	}

	s.setConfig(&newCfg)
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "配置更新成功",
		"reload":  reloadResult,
	})
}

// handleValidateConfig 验证配置合法性。
func (s *Server) handleValidateConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	if err := cfg.Validate(); err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

// handleGetMetricsURL 获取 metrics 端点 URL。
func (s *Server) handleGetMetricsURL(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	host := cfg.Prometheus.ListenAddress
	if host == "0.0.0.0" {
		host = "localhost"
	}
	port := cfg.Prometheus.ListenPort
	if port == 0 {
		port = 8080
	}
	url := fmt.Sprintf("http://%s:%d/metrics", host, port)
	s.writeJSON(w, http.StatusOK, map[string]string{
		"url": url,
	})
}

// handleTestMySQL 测试 MySQL 连接。
func (s *Server) handleTestMySQL(w http.ResponseWriter, r *http.Request) {
	var mysqlCfg config.MySQLConfig
	if err := json.NewDecoder(r.Body).Decode(&mysqlCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析 MySQL 配置失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := datasource.NewMySQLClient(mysqlCfg)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "MySQL 连接测试成功",
	})
}

// handleTestIoTDB 测试 IoTDB 连接。
func (s *Server) handleTestIoTDB(w http.ResponseWriter, r *http.Request) {
	var iotdbCfg config.IoTDBConfig
	if err := json.NewDecoder(r.Body).Decode(&iotdbCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析 IoTDB 配置失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := datasource.NewIoTDBClient(iotdbCfg)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	// 使用 show databases 来测试连接
	err = client.TestConnection(ctx)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("IoTDB 连接测试失败: %v", err),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "IoTDB 连接测试成功",
	})
}

// handleTestHTTPAPI 测试 HTTP API 连接。
func (s *Server) handleTestHTTPAPI(w http.ResponseWriter, r *http.Request) {
	var httpapiCfg config.HTTPAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&httpapiCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析 HTTP API 配置失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := datasource.NewHTTPAPIClient(httpapiCfg)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer client.Close()

	// 测试连接：尝试获取 JSON 根路径的值（如果响应是对象，会返回错误，但至少能验证连接）
	_, err = client.QueryScalar(ctx, "")
	if err != nil {
		// 如果根路径失败，尝试一个常见的测试路径
		_, err = client.QueryScalar(ctx, "status")
		if err != nil {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("HTTP API 连接测试失败: %v", err),
			})
			return
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "HTTP API 连接测试成功",
	})
}

// QueryPreviewRequest 查询预览请求。
type QueryPreviewRequest struct {
	Source       string             `json:"source"`
	Query        string             `json:"query"`
	Connection   string             `json:"connection,omitempty"`
	ResultField  string             `json:"result_field,omitempty"`
	MySQLConfig  *config.MySQLConfig  `json:"mysql_config,omitempty"`
	IoTDBConfig  *config.IoTDBConfig  `json:"iotdb_config,omitempty"`
	HTTPAPIConfig *config.HTTPAPIConfig `json:"http_api_config,omitempty"`
}

// handlePreviewQuery 预览 SQL 查询结果。
func (s *Server) handlePreviewQuery(w http.ResponseWriter, r *http.Request) {
	var req QueryPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var value float64
	var err error

	switch req.Source {
	case "mysql":
		var client *datasource.MySQLClient
		if req.MySQLConfig != nil {
			// 使用请求中的配置
			client, err = datasource.NewMySQLClient(*req.MySQLConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 MySQL 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
			// 使用已配置的连接
			cfg := s.getConfig()
			connName := req.Connection
			if connName == "" {
				connName = "default"
			}
			mysqlCfg, ok := cfg.MySQLConfigFor(connName)
			if !ok {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("MySQL 连接 %s 未配置", connName))
				return
			}
			client, err = datasource.NewMySQLClient(mysqlCfg)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 MySQL 客户端失败: %v", err))
				return
			}
			defer client.Close()
		}
		value, err = client.QueryScalar(ctx, req.Query)
	case "iotdb":
		var client *datasource.IoTDBClient
		if req.IoTDBConfig != nil {
			// 使用请求中的配置
			client, err = datasource.NewIoTDBClient(*req.IoTDBConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 IoTDB 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
			// 使用已配置的连接
			cfg := s.getConfig()
			client, err = datasource.NewIoTDBClient(cfg.IoTDB)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 IoTDB 客户端失败: %v", err))
				return
			}
			defer client.Close()
		}
		value, err = client.QueryScalar(ctx, req.Query, req.ResultField)
	case "http_api":
		var client *datasource.HTTPAPIClient
		if req.HTTPAPIConfig != nil {
			// 使用请求中的配置
			client, err = datasource.NewHTTPAPIClient(*req.HTTPAPIConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 HTTP API 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
			// 使用已配置的连接
			cfg := s.getConfig()
			connName := req.Connection
			if connName == "" {
				connName = "default"
			}
			httpapiCfg, ok := cfg.HTTPAPIConfigFor(connName)
			if !ok {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("HTTP API 连接 %s 未配置", connName))
				return
			}
			client, err = datasource.NewHTTPAPIClient(httpapiCfg)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 HTTP API 客户端失败: %v", err))
				return
			}
			defer client.Close()
		}
		// 对于 HTTP API，ResultField 存储 JSON 路径
		// Query 字段存储 URL（可选，如果为空则使用连接配置的 URL）
		jsonPath := req.ResultField
		if jsonPath == "" {
			s.writeError(w, http.StatusBadRequest, "HTTP API 预览需要指定 result_field（JSON 路径）")
			return
		}
		// 使用 req.Query 作为 URL（如果提供），否则使用连接配置的 URL
		queryURL := req.Query
		value, err = client.QueryScalar(ctx, jsonPath, queryURL)
	default:
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("不支持的数据源: %s", req.Source))
		return
	}

	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"value":   value,
	})
}

// handleListMetrics 获取所有指标列表。
func (s *Server) handleListMetrics(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	s.writeJSON(w, http.StatusOK, cfg.Metrics)
}

// handleGetMetric 获取单个指标详情。
func (s *Server) handleGetMetric(w http.ResponseWriter, r *http.Request) {
	metricName := strings.TrimPrefix(r.URL.Path, "/api/metrics/")
	metricName = strings.TrimSuffix(metricName, "/")
	cfg := s.getConfig()
	for _, metric := range cfg.Metrics {
		if metric.Name == metricName {
			s.writeJSON(w, http.StatusOK, metric)
			return
		}
	}
	s.writeError(w, http.StatusNotFound, "指标未找到")
}

// handleCreateMetric 创建新指标。
func (s *Server) handleCreateMetric(w http.ResponseWriter, r *http.Request) {
	var metric config.MetricSpec
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析指标配置失败: %v", err))
		return
	}

	cfg := s.getConfig()
	// 检查是否已存在
	for _, m := range cfg.Metrics {
		if m.Name == metric.Name {
			s.writeError(w, http.StatusConflict, "指标已存在")
			return
		}
	}

	cfg.Metrics = append(cfg.Metrics, metric)
	if err := cfg.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	if err := cfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	reloadResult := s.service.ReloadConfig(cfg)
	if !reloadResult.Success {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("热更新失败: %s", reloadResult.Error))
		return
	}

	s.setConfig(cfg)
	s.writeJSON(w, http.StatusCreated, metric)
}

// handleUpdateMetric 更新指标。
func (s *Server) handleUpdateMetric(w http.ResponseWriter, r *http.Request) {
	metricName := strings.TrimPrefix(r.URL.Path, "/api/metrics/")
	metricName = strings.TrimSuffix(metricName, "/")
	var metric config.MetricSpec
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析指标配置失败: %v", err))
		return
	}

	if metric.Name != metricName {
		s.writeError(w, http.StatusBadRequest, "指标名称不匹配")
		return
	}

	cfg := s.getConfig()
	found := false
	for i, m := range cfg.Metrics {
		if m.Name == metricName {
			cfg.Metrics[i] = metric
			found = true
			break
		}
	}

	if !found {
		s.writeError(w, http.StatusNotFound, "指标未找到")
		return
	}

	if err := cfg.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	if err := cfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	reloadResult := s.service.ReloadConfig(cfg)
	if !reloadResult.Success {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("热更新失败: %s", reloadResult.Error))
		return
	}

	s.setConfig(cfg)
	s.writeJSON(w, http.StatusOK, metric)
}

// handleDeleteMetric 删除指标。
func (s *Server) handleDeleteMetric(w http.ResponseWriter, r *http.Request) {
	metricName := strings.TrimPrefix(r.URL.Path, "/api/metrics/")
	metricName = strings.TrimSuffix(metricName, "/")
	cfg := s.getConfig()
	found := false
	for i, m := range cfg.Metrics {
		if m.Name == metricName {
			cfg.Metrics = append(cfg.Metrics[:i], cfg.Metrics[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		s.writeError(w, http.StatusNotFound, "指标未找到")
		return
	}

	if err := cfg.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	if err := cfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	reloadResult := s.service.ReloadConfig(cfg)
	if !reloadResult.Success {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("热更新失败: %s", reloadResult.Error))
		return
	}

	s.setConfig(cfg)
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "指标已删除"})
}
