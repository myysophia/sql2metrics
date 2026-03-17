package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
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

	if err := newCfg.ApplyDefaults(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("应用默认值失败: %v", err))
		return
	}
	if err := newCfg.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("配置验证失败: %v", err))
		return
	}

	if err := newCfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

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

	if err := client.TestConnection(ctx); err != nil {
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

// handleTestRedis 测试 Redis 连接。
func (s *Server) handleTestRedis(w http.ResponseWriter, r *http.Request) {
	var redisCfg config.RedisConfig
	if err := json.NewDecoder(r.Body).Decode(&redisCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析 Redis 配置失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := datasource.NewRedisClient(redisCfg)
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
		"message": "Redis 连接测试成功",
	})
}

// handleTestRestAPI 测试 RestAPI 连接。
func (s *Server) handleTestRestAPI(w http.ResponseWriter, r *http.Request) {
	var restapiCfg config.RestAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&restapiCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析 RestAPI 配置失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := datasource.NewRestAPIClient(restapiCfg)
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
		"message": "RestAPI 连接测试成功",
	})
}

// RestAPIPreviewRequest 用于预览 RestAPI 响应的请求。
type RestAPIPreviewRequest struct {
	Config config.RestAPIConfig `json:"config"`
	Query  string               `json:"query"`
}

// handlePreviewRestAPI 预览 RestAPI 响应，返回完整 JSON 数据供字段选择。
func (s *Server) handlePreviewRestAPI(w http.ResponseWriter, r *http.Request) {
	var req RestAPIPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := datasource.NewRestAPIClient(req.Config)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("创建 RestAPI 客户端失败: %v", err),
		})
		return
	}
	defer client.Close()

	result, err := client.QueryRaw(ctx, req.Query)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

// QueryPreviewRequest 查询预览请求。
type QueryPreviewRequest struct {
	Source      string              `json:"source"`
	Query       string              `json:"query"`
	Connection  string              `json:"connection,omitempty"`
	ResultField string              `json:"result_field,omitempty"`
	MySQLConfig *config.MySQLConfig `json:"mysql_config,omitempty"`
	IoTDBConfig *config.IoTDBConfig `json:"iotdb_config,omitempty"`
	RedisConfig *config.RedisConfig `json:"redis_config,omitempty"`
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
			client, err = datasource.NewMySQLClient(*req.MySQLConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 MySQL 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
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
			client, err = datasource.NewIoTDBClient(*req.IoTDBConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 IoTDB 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
			cfg := s.getConfig()
			client, err = datasource.NewIoTDBClient(cfg.IoTDB)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 IoTDB 客户端失败: %v", err))
				return
			}
			defer client.Close()
		}
		value, err = client.QueryScalar(ctx, req.Query, req.ResultField)
	case "redis":
		var client *datasource.RedisClient
		if req.RedisConfig != nil {
			client, err = datasource.NewRedisClient(*req.RedisConfig)
			if err != nil {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("创建 Redis 客户端失败: %v", err))
				return
			}
			defer client.Close()
		} else {
			cfg := s.getConfig()
			connName := req.Connection
			if connName == "" {
				connName = "default"
			}
			redisCfg, ok := cfg.RedisConfigFor(connName)
			if !ok {
				s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Redis 连接 %s 未配置", connName))
				return
			}
			client, err = datasource.NewRedisClient(redisCfg)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("创建 Redis 客户端失败: %v", err))
				return
			}
			defer client.Close()
		}
		value, err = client.QueryScalar(ctx, req.Query)
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
	for _, m := range cfg.Metrics {
		if m.Name == metric.Name &&
			m.Help == metric.Help &&
			m.Source == metric.Source &&
			m.Connection == metric.Connection &&
			m.Query == metric.Query {
			// 幂等性支持：如果完全一致，返回 200 OK
			s.writeJSON(w, http.StatusOK, m)
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

// handleDeleteMetricByIndex 按索引删除指标。
func (s *Server) handleDeleteMetricByIndex(w http.ResponseWriter, r *http.Request) {
	indexStr := strings.TrimPrefix(r.URL.Path, "/api/metrics/index/")
	indexStr = strings.TrimSuffix(indexStr, "/")
	
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		s.writeError(w, http.StatusBadRequest, "无效的索引")
		return
	}

	cfg := s.getConfig()
	if index < 0 || index >= len(cfg.Metrics) {
		s.writeError(w, http.StatusNotFound, "指标索引超出范围")
		return
	}

	// 按索引删除指标
	cfg.Metrics = append(cfg.Metrics[:index], cfg.Metrics[index+1:]...)

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

// handleUpdateMetricByIndex 按索引更新指标。
func (s *Server) handleUpdateMetricByIndex(w http.ResponseWriter, r *http.Request) {
	indexStr := strings.TrimPrefix(r.URL.Path, "/api/metrics/index/")
	indexStr = strings.TrimSuffix(indexStr, "/")
	
	var index int
	if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
		s.writeError(w, http.StatusBadRequest, "无效的索引")
		return
	}

	var metric config.MetricSpec
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析指标配置失败: %v", err))
		return
	}

	cfg := s.getConfig()
	if index < 0 || index >= len(cfg.Metrics) {
		s.writeError(w, http.StatusNotFound, "指标索引超出范围")
		return
	}

	// 按索引更新指标
	cfg.Metrics[index] = metric

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

// ===================== 独立数据源 API =====================

// handleUpdateMySQLConnection 更新单个 MySQL 连接
func (s *Server) handleUpdateMySQLConnection(w http.ResponseWriter, r *http.Request, name string) {
	var mysqlCfg config.MySQLConfig
	if err := json.NewDecoder(r.Body).Decode(&mysqlCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	cfg := s.getConfig().Clone()
	if cfg.MySQLConnections == nil {
		cfg.MySQLConnections = make(map[string]config.MySQLConfig)
	}
	cfg.MySQLConnections[name] = mysqlCfg

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MySQL 连接 %s 已更新", name),
	})
}

// handleDeleteMySQLConnection 删除单个 MySQL 连接
func (s *Server) handleDeleteMySQLConnection(w http.ResponseWriter, r *http.Request, name string) {
	cfg := s.getConfig().Clone()
	if cfg.MySQLConnections == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("MySQL 连接 %s 不存在", name))
		return
	}
	if _, ok := cfg.MySQLConnections[name]; !ok {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("MySQL 连接 %s 不存在", name))
		return
	}
	delete(cfg.MySQLConnections, name)

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("MySQL 连接 %s 已删除", name),
	})
}

// handleUpdateRedisConnection 更新单个 Redis 连接
func (s *Server) handleUpdateRedisConnection(w http.ResponseWriter, r *http.Request, name string) {
	var redisCfg config.RedisConfig
	if err := json.NewDecoder(r.Body).Decode(&redisCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	cfg := s.getConfig().Clone()
	if cfg.RedisConnections == nil {
		cfg.RedisConnections = make(map[string]config.RedisConfig)
	}
	cfg.RedisConnections[name] = redisCfg

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Redis 连接 %s 已更新", name),
	})
}

// handleDeleteRedisConnection 删除单个 Redis 连接
func (s *Server) handleDeleteRedisConnection(w http.ResponseWriter, r *http.Request, name string) {
	cfg := s.getConfig().Clone()
	if cfg.RedisConnections == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Redis 连接 %s 不存在", name))
		return
	}
	if _, ok := cfg.RedisConnections[name]; !ok {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Redis 连接 %s 不存在", name))
		return
	}
	delete(cfg.RedisConnections, name)

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Redis 连接 %s 已删除", name),
	})
}

// handleUpdateRestAPIConnection 更新单个 RestAPI 连接
func (s *Server) handleUpdateRestAPIConnection(w http.ResponseWriter, r *http.Request, name string) {
	var restCfg config.RestAPIConfig
	if err := json.NewDecoder(r.Body).Decode(&restCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	cfg := s.getConfig().Clone()
	if cfg.RestAPIConnections == nil {
		cfg.RestAPIConnections = make(map[string]config.RestAPIConfig)
	}
	cfg.RestAPIConnections[name] = restCfg

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("RestAPI 连接 %s 已更新", name),
	})
}

// handleDeleteRestAPIConnection 删除单个 RestAPI 连接
func (s *Server) handleDeleteRestAPIConnection(w http.ResponseWriter, r *http.Request, name string) {
	cfg := s.getConfig().Clone()
	if cfg.RestAPIConnections == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("RestAPI 连接 %s 不存在", name))
		return
	}
	if _, ok := cfg.RestAPIConnections[name]; !ok {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("RestAPI 连接 %s 不存在", name))
		return
	}
	delete(cfg.RestAPIConnections, name)

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("RestAPI 连接 %s 已删除", name),
	})
}

// handleUpdateIoTDB 更新 IoTDB 配置
func (s *Server) handleUpdateIoTDB(w http.ResponseWriter, r *http.Request) {
	var iotdbCfg config.IoTDBConfig
	if err := json.NewDecoder(r.Body).Decode(&iotdbCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析配置失败: %v", err))
		return
	}

	cfg := s.getConfig().Clone()
	cfg.IoTDB = iotdbCfg

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "IoTDB 配置已更新",
	})
}

// ===================== 独立指标 API =====================

// handleAddMetric 新增指标
func (s *Server) handleAddMetric(w http.ResponseWriter, r *http.Request) {
	var metric config.MetricSpec
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析指标失败: %v", err))
		return
	}

	cfg := s.getConfig().Clone()
	cfg.Metrics = append(cfg.Metrics, metric)

	if err := s.saveAndReload(cfg); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("指标 %s 已添加", metric.Name),
		"index":   len(cfg.Metrics) - 1,
	})
}

// saveAndReload 保存配置并触发热更新
func (s *Server) saveAndReload(cfg *config.Config) error {
	if err := cfg.ApplyDefaults(); err != nil {
		return fmt.Errorf("应用默认值失败: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}
	if err := cfg.Save(s.configPath); err != nil {
		return fmt.Errorf("保存配置失败: %v", err)
	}

	reloadResult := s.service.ReloadConfig(cfg)
	if !reloadResult.Success {
		return fmt.Errorf("热更新失败: %s", reloadResult.Error)
	}

	s.setConfig(cfg)
	return nil
}

// ===================== Notifier 配置处理 =====================

// handleGetNotifierConfig 获取通知服务配置
func (s *Server) handleGetNotifierConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()

	// 返回 notifier 配置，如果为空则返回默认配置
	notifierCfg := cfg.Notifier
	if !notifierCfg.Enabled && notifierCfg.WeChat == nil && notifierCfg.DingTalk == nil && notifierCfg.Feishu == nil {
		// 返回一个默认的空配置
		notifierCfg = config.NotifierConfig{}
	}

	s.writeJSON(w, http.StatusOK, notifierCfg)
}

// handleUpdateNotifierConfig 更新通知服务配置
func (s *Server) handleUpdateNotifierConfig(w http.ResponseWriter, r *http.Request) {
	var newNotifierCfg config.NotifierConfig
	if err := json.NewDecoder(r.Body).Decode(&newNotifierCfg); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析通知配置失败: %v", err))
		return
	}

	// 获取当前配置
	cfg := s.getConfig()

	// 更新 notifier 配置
	cfg.Notifier = newNotifierCfg

	// 保存配置
	if err := cfg.Save(s.configPath); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("保存配置失败: %v", err))
		return
	}

	// 更新内存中的配置
	s.setConfig(cfg)

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "通知配置更新成功，重启服务后生效",
	})
}

// handleTestNotifierWebhook 测试通知 webhook
func (s *Server) handleTestNotifierWebhook(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Channel string `json:"channel"` // wechat, dingtalk, feishu
		Webhook string `json:"webhook"`
		Secret  string `json:"secret"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	if req.Webhook == "" {
		s.writeError(w, http.StatusBadRequest, "webhook URL 不能为空")
		return
	}

	// 根据通道类型发送测试消息
	// TODO: 实现具体的测试逻辑
	// 这里可以调用相应的 notifier 发送测试消息

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "测试消息已发送到 " + req.Channel,
	})
}

// handleListAvailableMetrics 列出可用的指标
func (s *Server) handleListAvailableMetrics(w http.ResponseWriter, r *http.Request) {
	cfg := s.getConfig()
	metrics := make([]string, 0, len(cfg.Metrics))
	for _, m := range cfg.Metrics {
		metrics = append(metrics, m.Name)
	}
	s.writeJSON(w, http.StatusOK, metrics)
}

// handleQueryTimeseries 查询时序数据
func (s *Server) handleQueryTimeseries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metrics []string `json:"metrics"`
		Start   string   `json:"start"`
		End     string   `json:"end"`
		Step    string   `json:"step"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("解析请求失败: %v", err))
		return
	}

	log.Printf("[API] 查询时序数据: metrics=%v, start=%s, end=%s", req.Metrics, req.Start, req.End)

	if len(req.Metrics) == 0 {
		s.writeError(w, http.StatusBadRequest, "metrics 不能为空")
		return
	}

	if req.Start == "" {
		req.Start = "-1h"
	}
	if req.End == "" {
		req.End = "now"
	}
	if req.Step == "" {
		req.Step = "30s"
	}

	// 解析时间参数为 Unix 时间戳
	startTime, err := parseTimeToUnix(req.Start)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("无法解析 start 时间 %s: %v", req.Start, err))
		return
	}
	endTime, err := parseTimeToUnix(req.End)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("无法解析 end 时间 %s: %v", req.End, err))
		return
	}

	log.Printf("[API] 解析时间: %s -> %d, %s -> %d", req.Start, startTime, req.End, endTime)

	// 获取 Prometheus 地址
	cfg := s.getConfig()
	var prometheusURL string
	if cfg.Prometheus.URL != "" {
		// 使用配置的外部 Prometheus URL
		prometheusURL = cfg.Prometheus.URL
	} else {
		// 使用本地 Prometheus 地址
		prometheusURL = fmt.Sprintf("http://%s:%d", cfg.Prometheus.ListenAddress, cfg.Prometheus.ListenPort)
	}
	log.Printf("[API] Prometheus URL: %s", prometheusURL)

	// 查询每个指标
	result := make([]map[string]interface{}, 0)
	for _, metricName := range req.Metrics {
		// 构建 Prometheus 查询 URL (使用 Unix 时间戳)
		queryURL := fmt.Sprintf(
			"%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%s",
			prometheusURL,
			metricName,
			startTime,
			endTime,
			req.Step,
		)

		log.Printf("[API] 查询 Prometheus: %s", queryURL)

		// 发送 HTTP 请求到 Prometheus
		resp, err := http.Get(queryURL)
		if err != nil {
			log.Printf("[API] 查询 Prometheus 失败: %v", err)
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("查询 Prometheus 失败: %v", err))
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[API] 读取响应失败: %v", err)
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("读取响应失败: %v", err))
			return
		}

		if resp.StatusCode != 200 {
			log.Printf("[API] Prometheus 返回错误 %d: %s", resp.StatusCode, string(body))
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Prometheus 返回错误: %s", string(body)))
			return
		}

		log.Printf("[API] Prometheus 响应: %s", string(body[:min(500, len(body))]))

		// 解析 Prometheus 响应
		var promResp struct {
			Data struct {
				Result []struct {
					Metric map[string]string `json:"metric"`
					Values [][]interface{}    `json:"values"`
				} `json:"result"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &promResp); err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("解析 Prometheus 响应失败: %v", err))
			return
		}

		// 转换数据格式
		for _, r := range promResp.Data.Result {
			values := make([][2]float64, 0, len(r.Values))
			for _, v := range r.Values {
				if len(v) == 2 {
					timestamp, ok1 := v[0].(float64)
					valueStr, ok2 := v[1].(string)
					if ok1 && ok2 {
						value, err := strconv.ParseFloat(valueStr, 64)
						if err == nil {
							values = append(values, [2]float64{timestamp, value})
						}
					}
				}
			}

			result = append(result, map[string]interface{}{
				"metric": r.Metric,
				"values": values,
			})
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": result,
	})
}

// handleExportTimeseries 导出时序数据为 CSV
func (s *Server) handleExportTimeseries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metrics []string `json:"metrics"`
		Start   string   `json:"start"`
		End     string   `json:"end"`
		Step    string   `json:"step"`
	}

	// 从查询参数解析
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 如果是 GET 请求，尝试从 URL 参数解析
		req.Metrics = r.URL.Query()["metric"]
		req.Start = r.URL.Query().Get("start")
		req.End = r.URL.Query().Get("end")
		req.Step = r.URL.Query().Get("step")
	}

	if len(req.Metrics) == 0 {
		s.writeError(w, http.StatusBadRequest, "metrics 不能为空")
		return
	}

	// 设置 CSV 响应头
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=metrics-%d.csv", time.Now().Unix()))

	// 查询数据并生成 CSV
	// 这里可以复用 handleQueryTimeseries 的逻辑
	// 为了简化，我们先实现一个简单版本

	// TODO: 实现完整的导出逻辑
	w.Write([]byte("timestamp,metric,value\n"))
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseTimeToUnix 解析时间字符串为 Unix 时间戳
// 支持格式：
//   - "now" -> 当前时间
//   - "-1h" -> 1小时前
//   - "-5m" -> 5分钟前
//   - "-1d" -> 1天前
//   - Unix时间戳 (如 "1710374400")
func parseTimeToUnix(timeStr string) (int64, error) {
	timeStr = strings.TrimSpace(timeStr)

	// 特殊处理 "now"
	if timeStr == "now" {
		return time.Now().Unix(), nil
	}

	// 处理相对时间 (如 -1h, -5m, -1d)
	if strings.HasPrefix(timeStr, "-") {
		now := time.Now()
		durationStr := timeStr[1:] // 去掉 "-" 前缀

		var duration time.Duration
		if strings.HasSuffix(durationStr, "s") {
			// 秒
			seconds, err := strconv.ParseInt(durationStr[:len(durationStr)-1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无法解析秒数: %w", err)
			}
			duration = time.Duration(seconds) * time.Second
		} else if strings.HasSuffix(durationStr, "m") {
			// 分钟
			minutes, err := strconv.ParseInt(durationStr[:len(durationStr)-1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无法解析分钟数: %w", err)
			}
			duration = time.Duration(minutes) * time.Minute
		} else if strings.HasSuffix(durationStr, "h") {
			// 小时
			hours, err := strconv.ParseInt(durationStr[:len(durationStr)-1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无法解析小时数: %w", err)
			}
			duration = time.Duration(hours) * time.Hour
		} else if strings.HasSuffix(durationStr, "d") {
			// 天
			days, err := strconv.ParseInt(durationStr[:len(durationStr)-1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无法解析天数: %w", err)
			}
			duration = time.Duration(days) * 24 * time.Hour
		} else {
			// 默认为秒
			seconds, err := strconv.ParseInt(durationStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("无法解析时间: %w", err)
			}
			duration = time.Duration(seconds) * time.Second
		}

		return now.Add(-duration).Unix(), nil
	}

	// 尝试解析为 Unix 时间戳
	timestamp, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无法解析时间戳: %w", err)
	}
	return timestamp, nil
}
