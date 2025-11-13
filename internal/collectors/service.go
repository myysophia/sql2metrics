package collectors

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/datasource"
)

// Service 负责调度查询并更新 Prometheus 指标。
type Service struct {
	cfg        *config.Config
	mysql      map[string]*datasource.MySQLClient
	iotdb      *datasource.IoTDBClient
	httpapi    map[string]*datasource.HTTPAPIClient
	metrics    []metricHolder
	errorCount prometheus.Counter
	lastRun    prometheus.Gauge
	registry   *prometheus.Registry
	mu         sync.RWMutex
}

type metricHolder struct {
	spec  config.MetricSpec
	gauge prometheus.Gauge
}

// NewService 构造采集服务，按需初始化数据源。
// 注意：即使某些数据源连接失败，服务也会成功创建，只是相关指标无法采集。
func NewService(cfg *config.Config) (*Service, error) {
	svc := &Service{
		cfg:      cfg,
		mysql:    make(map[string]*datasource.MySQLClient),
		httpapi:  make(map[string]*datasource.HTTPAPIClient),
		registry: prometheus.NewRegistry(),
	}
	
	// 初始化 IoTDB 连接（失败时只记录警告，不阻止服务启动）
	if needsSource(cfg.Metrics, "iotdb") {
		iotdbClient, err := datasource.NewIoTDBClient(cfg.IoTDB)
		if err != nil {
			log.Printf("警告: IoTDB 连接失败，相关指标将无法采集: %v", err)
		} else {
			svc.iotdb = iotdbClient
		}
	}

	// 初始化 MySQL 连接（失败时只记录警告，不阻止服务启动）
	for connName := range mysqlConnectionsNeeded(cfg) {
		mysqlCfg, ok := cfg.MySQLConfigFor(connName)
		if !ok {
			log.Printf("警告: 未找到 MySQL 连接配置 %s，相关指标将无法采集", connName)
			continue
		}
		client, err := datasource.NewMySQLClient(mysqlCfg)
		if err != nil {
			log.Printf("警告: MySQL 连接 %s 失败，相关指标将无法采集: %v", connName, err)
		} else {
			svc.mysql[connName] = client
		}
	}

	// 初始化 HTTP API 连接（失败时只记录警告，不阻止服务启动）
	for connName := range httpAPIConnectionsNeeded(cfg) {
		httpapiCfg, ok := cfg.HTTPAPIConfigFor(connName)
		if !ok {
			log.Printf("警告: 未找到 HTTP API 连接配置 %s，相关指标将无法采集", connName)
			continue
		}
		client, err := datasource.NewHTTPAPIClient(httpapiCfg)
		if err != nil {
			log.Printf("警告: HTTP API 连接 %s 失败，相关指标将无法采集: %v", connName, err)
		} else {
			svc.httpapi[connName] = client
		}
	}

	for _, spec := range cfg.Metrics {
		metricType := spec.Type
		if metricType == "" {
			metricType = "gauge"
		}

		var metric prometheus.Collector
		switch metricType {
		case "gauge":
			metric = prometheus.NewGauge(prometheus.GaugeOpts{
				Name:        spec.Name,
				Help:        spec.Help,
				ConstLabels: spec.Labels,
			})
		case "counter":
			metric = prometheus.NewCounter(prometheus.CounterOpts{
				Name:        spec.Name,
				Help:        spec.Help,
				ConstLabels: spec.Labels,
			})
		case "histogram":
			buckets := spec.Buckets
			if len(buckets) == 0 {
				buckets = prometheus.DefBuckets
			}
			metric = prometheus.NewHistogram(prometheus.HistogramOpts{
				Name:        spec.Name,
				Help:        spec.Help,
				ConstLabels: spec.Labels,
				Buckets:     buckets,
			})
		case "summary":
			objectives := spec.Objectives
			if len(objectives) == 0 {
				objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
			}
			metric = prometheus.NewSummary(prometheus.SummaryOpts{
				Name:        spec.Name,
				Help:        spec.Help,
				ConstLabels: spec.Labels,
				Objectives:  objectives,
			})
		default:
			return nil, fmt.Errorf("不支持的指标类型: %s", metricType)
		}

		if err := svc.registry.Register(metric); err != nil {
			return nil, fmt.Errorf("注册指标 %s 失败: %w", spec.Name, err)
		}

		// 目前只支持 Gauge 类型的更新，其他类型需要不同的更新逻辑
		if gauge, ok := metric.(prometheus.Gauge); ok {
			svc.metrics = append(svc.metrics, metricHolder{
				spec:  spec,
				gauge: gauge,
			})
		}
	}

	svc.errorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "collector_errors_total",
		Help: "采集周期内出现错误的次数",
	})
	svc.lastRun = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "collector_last_success_timestamp_seconds",
		Help: "最近一次成功采集的 Unix 时间戳",
	})
	svc.registry.MustRegister(svc.errorCount, svc.lastRun)
	
	// 同时注册到默认注册表以保持兼容性
	prometheus.MustRegister(svc.errorCount, svc.lastRun)
	for _, holder := range svc.metrics {
		prometheus.MustRegister(holder.gauge)
	}
	
	return svc, nil
}

func needsSource(metrics []config.MetricSpec, source string) bool {
	for _, m := range metrics {
		if m.Source == source {
			return true
		}
	}
	return false
}

func mysqlConnectionsNeeded(cfg *config.Config) map[string]struct{} {
	required := make(map[string]struct{})
	for _, m := range cfg.Metrics {
		if m.Source != "mysql" {
			continue
		}
		name := m.Connection
		if name == "" {
			name = "default"
		}
		required[name] = struct{}{}
	}
	return required
}

func httpAPIConnectionsNeeded(cfg *config.Config) map[string]struct{} {
	required := make(map[string]struct{})
	for _, m := range cfg.Metrics {
		if m.Source != "http_api" {
			continue
		}
		name := m.Connection
		if name == "" {
			name = "default"
		}
		required[name] = struct{}{}
	}
	return required
}

// Run 启动周期性采集流程。
func (s *Service) Run(ctx context.Context) {
	interval, err := s.cfg.Schedule.IntervalDuration()
	if err != nil {
		log.Printf("解析采集周期失败: %v", err)
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.execute(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.execute(ctx)
		}
	}
}

func (s *Service) execute(ctx context.Context) {
	log.Printf("开始执行采集周期，共 %d 个指标", len(s.metrics))
	var success bool
	for _, holder := range s.metrics {
		start := time.Now()
		log.Printf("开始更新指标 %s (source=%s)", holder.spec.Name, holder.spec.Source)
		value, err := s.queryMetric(ctx, holder.spec)
		if err != nil {
			log.Printf("更新指标 %s 失败: %v", holder.spec.Name, err)
			holder.gauge.Set(math.NaN())
			s.errorCount.Inc()
			continue
		}
		holder.gauge.Set(value)
		success = true
		log.Printf("指标 %s 更新成功，值=%.3f，耗时=%s", holder.spec.Name, value, time.Since(start))
	}
	if success {
		s.lastRun.Set(float64(time.Now().Unix()))
		log.Printf("采集周期完成")
	} else {
		log.Printf("采集周期无成功指标，请检查数据源或配置")
	}
}

func (s *Service) queryMetric(ctx context.Context, spec config.MetricSpec) (float64, error) {
	switch spec.Source {
	case "mysql":
		conn := spec.Connection
		if conn == "" {
			conn = "default"
		}
		client, ok := s.mysql[conn]
		if !ok {
			return 0, fmt.Errorf("MySQL 连接 %s 未初始化", conn)
		}
		log.Printf("执行 MySQL 查询（连接=%s）: %s", conn, spec.Query)
		return client.QueryScalar(ctx, spec.Query)
	case "iotdb":
		if s.iotdb == nil {
			return 0, ErrDataSourceUnavailable(spec.Source)
		}
		log.Printf("执行 IoTDB 查询: %s", spec.Query)
		return s.iotdb.QueryScalar(ctx, spec.Query, spec.ResultField)
	case "http_api":
		conn := spec.Connection
		if conn == "" {
			conn = "default"
		}
		client, ok := s.httpapi[conn]
		if !ok {
			return 0, fmt.Errorf("HTTP API 连接 %s 未初始化", conn)
		}
		// 对于 HTTP API，Query 字段存储 URL（可选，如果为空则使用连接配置的 URL）
		// ResultField 存储 JSON 路径
		jsonPath := spec.ResultField
		if jsonPath == "" {
			return 0, fmt.Errorf("HTTP API 指标 %s 缺少 result_field（JSON 路径）", spec.Name)
		}
		log.Printf("执行 HTTP API 查询（连接=%s，JSON 路径=%s）", conn, jsonPath)
		return client.QueryScalar(ctx, jsonPath)
	default:
		return 0, ErrDataSourceUnavailable(spec.Source)
	}
}

func ErrDataSourceUnavailable(source string) error {
	return fmt.Errorf("数据源 %s 未准备就绪", source)
}

// Close 释放资源。
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mysql != nil {
		for name, client := range s.mysql {
			if err := client.Close(); err != nil {
				log.Printf("关闭 MySQL 连接 %s 失败: %v", name, err)
			}
		}
	}
	if s.iotdb != nil {
		if err := s.iotdb.Close(); err != nil {
			log.Printf("关闭 IoTDB 连接失败: %v", err)
		}
	}
	if s.httpapi != nil {
		for name, client := range s.httpapi {
			if err := client.Close(); err != nil {
				log.Printf("关闭 HTTP API 连接 %s 失败: %v", name, err)
			}
		}
	}
	// 注销指标
	if s.registry != nil {
		for _, holder := range s.metrics {
			s.registry.Unregister(holder.gauge)
			prometheus.Unregister(holder.gauge)
		}
		s.registry.Unregister(s.errorCount)
		s.registry.Unregister(s.lastRun)
		prometheus.Unregister(s.errorCount)
		prometheus.Unregister(s.lastRun)
	}
}

// GetRegistry 返回 Prometheus 注册表。
func (s *Service) GetRegistry() *prometheus.Registry {
	return s.registry
}

// GetPrometheusHandler 返回 Prometheus HTTP handler。
func (s *Service) GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// ReloadResult 热更新结果。
type ReloadResult struct {
	Success   bool     `json:"success"`
	Error     string   `json:"error,omitempty"`
	Message   string   `json:"message"`
	Metrics   []string `json:"metrics,omitempty"`
	Removed   []string `json:"removed,omitempty"`
}

// ReloadConfig 重新加载配置（热更新）。
func (s *Service) ReloadConfig(newCfg *config.Config) ReloadResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 收集需要移除的指标名称
	oldMetricNames := make(map[string]bool)
	for _, holder := range s.metrics {
		oldMetricNames[holder.spec.Name] = true
	}

	newMetricNames := make(map[string]bool)
	for _, spec := range newCfg.Metrics {
		newMetricNames[spec.Name] = true
	}

	// 找出需要移除的指标
	var removed []string
	for name := range oldMetricNames {
		if !newMetricNames[name] {
			removed = append(removed, name)
		}
	}

	// 注销旧指标
	for _, holder := range s.metrics {
		if !newMetricNames[holder.spec.Name] {
			s.registry.Unregister(holder.gauge)
			prometheus.Unregister(holder.gauge)
		}
	}

	// 关闭不再需要的数据源连接
	oldMySQLConnections := make(map[string]bool)
	for name := range s.mysql {
		oldMySQLConnections[name] = true
	}

	newMySQLConnections := mysqlConnectionsNeeded(newCfg)
	for name := range oldMySQLConnections {
		if _, needed := newMySQLConnections[name]; !needed {
			if client, ok := s.mysql[name]; ok {
				client.Close()
				delete(s.mysql, name)
			}
		}
	}

	// 关闭不再需要的 HTTP API 连接
	oldHTTPAPIConnections := make(map[string]bool)
	for name := range s.httpapi {
		oldHTTPAPIConnections[name] = true
	}

	newHTTPAPIConnections := httpAPIConnectionsNeeded(newCfg)
	for name := range oldHTTPAPIConnections {
		if _, needed := newHTTPAPIConnections[name]; !needed {
			if client, ok := s.httpapi[name]; ok {
				client.Close()
				delete(s.httpapi, name)
			}
		}
	}

	// 添加新的 HTTP API 连接
	for connName := range newHTTPAPIConnections {
		if _, exists := s.httpapi[connName]; !exists {
			httpapiCfg, ok := newCfg.HTTPAPIConfigFor(connName)
			if ok {
				client, err := datasource.NewHTTPAPIClient(httpapiCfg)
				if err != nil {
					log.Printf("警告: HTTP API 连接 %s 初始化失败: %v", connName, err)
				} else {
					s.httpapi[connName] = client
				}
			}
		}
	}

	// 检查是否需要 IoTDB
	needsIoTDB := needsSource(newCfg.Metrics, "iotdb")
	if !needsIoTDB && s.iotdb != nil {
		s.iotdb.Close()
		s.iotdb = nil
	} else if needsIoTDB && s.iotdb == nil {
		var err error
		s.iotdb, err = datasource.NewIoTDBClient(newCfg.IoTDB)
		if err != nil {
			return ReloadResult{
				Success: false,
				Error:   fmt.Sprintf("初始化 IoTDB 连接失败: %v", err),
				Message: "热更新失败",
			}
		}
	}

	// 创建新的 MySQL 连接（如果需要）
	for connName := range newMySQLConnections {
		if _, exists := s.mysql[connName]; !exists {
			mysqlCfg, ok := newCfg.MySQLConfigFor(connName)
			if !ok {
				return ReloadResult{
					Success: false,
					Error:   fmt.Sprintf("未找到 MySQL 连接 %s", connName),
					Message: "热更新失败",
				}
			}
			client, err := datasource.NewMySQLClient(mysqlCfg)
			if err != nil {
				return ReloadResult{
					Success: false,
					Error:   fmt.Sprintf("初始化 MySQL 连接 %s 失败: %v", connName, err),
					Message: "热更新失败",
				}
			}
			s.mysql[connName] = client
		}
	}

	// 注册新指标或更新现有指标
	var newMetrics []string
	var updatedMetrics []metricHolder

	for _, spec := range newCfg.Metrics {
		metricType := spec.Type
		if metricType == "" {
			metricType = "gauge"
		}

		// 检查是否已存在
		var existingHolder *metricHolder
		for i, holder := range s.metrics {
			if holder.spec.Name == spec.Name {
				existingHolder = &s.metrics[i]
				break
			}
		}

		if existingHolder != nil {
			// 更新现有指标（如果类型或标签改变，需要重新注册）
			if existingHolder.spec.Type != spec.Type || !labelsEqual(existingHolder.spec.Labels, spec.Labels) {
				// 注销旧指标
				s.registry.Unregister(existingHolder.gauge)
				prometheus.Unregister(existingHolder.gauge)

				// 创建新指标
				var metric prometheus.Collector
				switch metricType {
				case "gauge":
					metric = prometheus.NewGauge(prometheus.GaugeOpts{
						Name:        spec.Name,
						Help:        spec.Help,
						ConstLabels: spec.Labels,
					})
				case "counter":
					metric = prometheus.NewCounter(prometheus.CounterOpts{
						Name:        spec.Name,
						Help:        spec.Help,
						ConstLabels: spec.Labels,
					})
				case "histogram":
					buckets := spec.Buckets
					if len(buckets) == 0 {
						buckets = prometheus.DefBuckets
					}
					metric = prometheus.NewHistogram(prometheus.HistogramOpts{
						Name:        spec.Name,
						Help:        spec.Help,
						ConstLabels: spec.Labels,
						Buckets:     buckets,
					})
				case "summary":
					objectives := spec.Objectives
					if len(objectives) == 0 {
						objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
					}
					metric = prometheus.NewSummary(prometheus.SummaryOpts{
						Name:        spec.Name,
						Help:        spec.Help,
						ConstLabels: spec.Labels,
						Objectives:  objectives,
					})
				}

				if err := s.registry.Register(metric); err != nil {
					return ReloadResult{
						Success: false,
						Error:   fmt.Sprintf("注册指标 %s 失败: %v", spec.Name, err),
						Message: "热更新失败",
					}
				}

				if gauge, ok := metric.(prometheus.Gauge); ok {
					existingHolder.gauge = gauge
					existingHolder.spec = spec
					prometheus.MustRegister(gauge)
				}
			} else {
				// 只更新 spec
				existingHolder.spec = spec
			}
			updatedMetrics = append(updatedMetrics, *existingHolder)
		} else {
			// 创建新指标
			var metric prometheus.Collector
			switch metricType {
			case "gauge":
				metric = prometheus.NewGauge(prometheus.GaugeOpts{
					Name:        spec.Name,
					Help:        spec.Help,
					ConstLabels: spec.Labels,
				})
			case "counter":
				metric = prometheus.NewCounter(prometheus.CounterOpts{
					Name:        spec.Name,
					Help:        spec.Help,
					ConstLabels: spec.Labels,
				})
			case "histogram":
				buckets := spec.Buckets
				if len(buckets) == 0 {
					buckets = prometheus.DefBuckets
				}
				metric = prometheus.NewHistogram(prometheus.HistogramOpts{
					Name:        spec.Name,
					Help:        spec.Help,
					ConstLabels: spec.Labels,
					Buckets:     buckets,
				})
			case "summary":
				objectives := spec.Objectives
				if len(objectives) == 0 {
					objectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001}
				}
				metric = prometheus.NewSummary(prometheus.SummaryOpts{
					Name:        spec.Name,
					Help:        spec.Help,
					ConstLabels: spec.Labels,
					Objectives:  objectives,
				})
			}

			if err := s.registry.Register(metric); err != nil {
				return ReloadResult{
					Success: false,
					Error:   fmt.Sprintf("注册指标 %s 失败: %v", spec.Name, err),
					Message: "热更新失败",
				}
			}

			if gauge, ok := metric.(prometheus.Gauge); ok {
				holder := metricHolder{
					spec:  spec,
					gauge: gauge,
				}
				updatedMetrics = append(updatedMetrics, holder)
				prometheus.MustRegister(gauge)
				newMetrics = append(newMetrics, spec.Name)
			}
		}
	}

	// 更新 metrics 列表
	s.metrics = updatedMetrics
	s.cfg = newCfg

	var metricNames []string
	for _, m := range newMetrics {
		metricNames = append(metricNames, m)
	}

	return ReloadResult{
		Success: true,
		Message: "配置热更新成功",
		Metrics: metricNames,
		Removed: removed,
	}
}

func labelsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
