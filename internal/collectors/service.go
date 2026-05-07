package collectors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/company/ems-devices/internal/alerts"
	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/datasource"
)

// Service 负责调度查询并更新 Prometheus 指标。
type Service struct {
	cfg             *config.Config
	mysql           map[string]*datasource.MySQLClient
	redis           map[string]*datasource.RedisClient
	iotdb           *datasource.IoTDBClient
	restapi         map[string]*datasource.RestAPIClient
	metrics         []metricHolder
	errorCount      prometheus.Counter
	lastRun         prometheus.Gauge
	registry        *prometheus.Registry
	alertEvaluator  *alerts.Evaluator
	currentValues   map[string]float64 // Track current metric values for alerts
	mu              sync.RWMutex
}

type metricHolder struct {
	spec  config.MetricSpec
	gauge prometheus.Gauge
}

// NewService 构造采集服务，按需初始化数据源。
// 注意：即使某些数据源连接失败，服务也会成功创建，只是相关指标无法采集。
func NewService(cfg *config.Config) (*Service, error) {
	svc := &Service{
		cfg:           cfg,
		mysql:         make(map[string]*datasource.MySQLClient),
		redis:         make(map[string]*datasource.RedisClient),
		restapi:       make(map[string]*datasource.RestAPIClient),
		registry:      prometheus.NewRegistry(),
		currentValues: make(map[string]float64),
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

	// 初始化 Redis 连接（失败时只记录警告，不阻止服务启动）
	for connName := range redisConnectionsNeeded(cfg) {
		redisCfg, ok := cfg.RedisConfigFor(connName)
		if !ok {
			log.Printf("警告: 未找到 Redis 连接配置 %s，相关指标将无法采集", connName)
			continue
		}
		client, err := datasource.NewRedisClient(redisCfg)
		if err != nil {
			log.Printf("警告: Redis 连接 %s 失败，相关指标将无法采集: %v", connName, err)
		} else {
			svc.redis[connName] = client
		}
	}

	// 初始化 RestAPI 连接（失败时只记录警告，不阻止服务启动）
	for connName := range restapiConnectionsNeeded(cfg) {
		restapiCfg, ok := cfg.RestAPIConfigFor(connName)
		if !ok {
			log.Printf("警告: 未找到 RestAPI 连接配置 %s，相关指标将无法采集", connName)
			continue
		}
		client, err := datasource.NewRestAPIClient(restapiCfg)
		if err != nil {
			log.Printf("警告: RestAPI 连接 %s 失败，相关指标将无法采集: %v", connName, err)
		} else {
			svc.restapi[connName] = client
		}
	}
	for _, spec := range cfg.Metrics {
		if spec.Enabled != nil && !*spec.Enabled {
			continue
		}
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
	prometheus.DefaultRegisterer.MustRegister(svc.errorCount, svc.lastRun)
	for _, holder := range svc.metrics {
		prometheus.DefaultRegisterer.MustRegister(holder.gauge)
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

func redisConnectionsNeeded(cfg *config.Config) map[string]struct{} {
	required := make(map[string]struct{})
	for _, m := range cfg.Metrics {
		if m.Source != "redis" {
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

func restapiConnectionsNeeded(cfg *config.Config) map[string]struct{} {
	required := make(map[string]struct{})
	for _, m := range cfg.Metrics {
		if m.Source != "restapi" {
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
		if holder.spec.Enabled != nil && !*holder.spec.Enabled {
			continue
		}
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

		// 存储当前指标值供告警使用
		s.mu.Lock()
		s.currentValues[holder.spec.Name] = value
		s.mu.Unlock()

		// 保存指标值到告警存储
		if s.alertEvaluator != nil {
			s.alertEvaluator.MetricStore().AddValue(holder.spec.Name, value)
		}
	}
	if success {
		s.lastRun.Set(float64(time.Now().Unix()))
		log.Printf("采集周期完成")
	} else {
		log.Printf("采集周期无成功指标，请检查数据源或配置")
	}

	// 触发 collection 模式告警评估
	if s.alertEvaluator != nil {
		go s.alertEvaluator.EvaluateCollectionModeAlerts(ctx)
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
	case "redis":
		conn := spec.Connection
		if conn == "" {
			conn = "default"
		}
		client, ok := s.redis[conn]
		if !ok {
			return 0, fmt.Errorf("Redis 连接 %s 未初始化", conn)
		}
		log.Printf("执行 Redis 命令（连接=%s）: %s", conn, spec.Query)
		return client.QueryScalar(ctx, spec.Query)
	case "restapi":
		conn := spec.Connection
		if conn == "" {
			conn = "default"
		}
		client, ok := s.restapi[conn]
		if !ok {
			return 0, fmt.Errorf("RestAPI 连接 %s 未初始化", conn)
		}
		log.Printf("执行 RestAPI 查询（连接=%s）: %s", conn, spec.Query)
		return client.QueryScalar(ctx, spec.Query, spec.ResultField)
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
	if s.redis != nil {
		for name, client := range s.redis {
			if err := client.Close(); err != nil {
				log.Printf("关闭 Redis 连接 %s 失败: %v", name, err)
			}
		}
	}
	if s.iotdb != nil {
		if err := s.iotdb.Close(); err != nil {
			log.Printf("关闭 IoTDB 连接失败: %v", err)
		}
	}
	if s.restapi != nil {
		for name, client := range s.restapi {
			if err := client.Close(); err != nil {
				log.Printf("关闭 RestAPI 连接 %s 失败: %v", name, err)
			}
		}
	}
	if s.registry != nil {
		for _, holder := range s.metrics {
			s.registry.Unregister(holder.gauge)
			prometheus.DefaultRegisterer.Unregister(holder.gauge)
		}
		s.registry.Unregister(s.errorCount)
		s.registry.Unregister(s.lastRun)
			prometheus.DefaultRegisterer.Unregister(s.errorCount)
			prometheus.DefaultRegisterer.Unregister(s.lastRun)
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

// SetAlertEvaluator sets the alert evaluator
func (s *Service) SetAlertEvaluator(evaluator *alerts.Evaluator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertEvaluator = evaluator
}

// GetMetricValue returns the current value of a metric (implements alerts.Service interface)
func (s *Service) GetMetricValue(metricName string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.currentValues[metricName]
	if !ok {
		return math.NaN(), false
	}
	return value, true
}

// RunScheduledEvaluation runs the scheduled alert evaluation loop
func (s *Service) RunScheduledEvaluation(ctx context.Context) {
	if s.alertEvaluator == nil {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Printf("启动告警定时评估循环")

	for {
		select {
		case <-ctx.Done():
			log.Printf("告警定时评估循环已停止")
			return
		case <-ticker.C:
			s.alertEvaluator.EvaluateScheduledModeAlerts(ctx)
		}
	}
}

// ReloadResult 热更新结果。
type ReloadResult struct {
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
	Message string   `json:"message"`
	Metrics []string `json:"metrics,omitempty"`
	Removed []string `json:"removed,omitempty"`
}

// ReloadConfig 重新加载配置（热更新）。
func (s *Service) ReloadConfig(newCfg *config.Config) ReloadResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldCfg := s.cfg

	oldMetricNames := make(map[string]bool)
	for _, holder := range s.metrics {
		oldMetricNames[holder.spec.Name] = true
	}

	newMetricNames := make(map[string]bool)
	for _, spec := range newCfg.Metrics {
		newMetricNames[spec.Name] = true
	}

	var removed []string
	for name := range oldMetricNames {
		if !newMetricNames[name] {
			removed = append(removed, name)
		}
	}

	for _, holder := range s.metrics {
		if !newMetricNames[holder.spec.Name] {
			s.registry.Unregister(holder.gauge)
			prometheus.DefaultRegisterer.Unregister(holder.gauge)
		}
	}

	oldMySQLConnections := make(map[string]bool)
	for name := range s.mysql {
		oldMySQLConnections[name] = true
	}
	oldRedisConnections := make(map[string]bool)
	for name := range s.redis {
		oldRedisConnections[name] = true
	}
	oldRestAPIConnections := make(map[string]bool)
	for name := range s.restapi {
		oldRestAPIConnections[name] = true
	}

	newMySQLConnections := mysqlConnectionsNeeded(newCfg)
	newRedisConnections := redisConnectionsNeeded(newCfg)
	newRestAPIConnections := restapiConnectionsNeeded(newCfg)

	for name := range oldMySQLConnections {
		if _, needed := newMySQLConnections[name]; !needed {
			if client, ok := s.mysql[name]; ok {
				client.Close()
				delete(s.mysql, name)
			}
		}
	}
	for name := range oldRedisConnections {
		if _, needed := newRedisConnections[name]; !needed {
			if client, ok := s.redis[name]; ok {
				client.Close()
				delete(s.redis, name)
			}
		}
	}
	for name := range oldRestAPIConnections {
		if _, needed := newRestAPIConnections[name]; !needed {
			if client, ok := s.restapi[name]; ok {
				client.Close()
				delete(s.restapi, name)
			}
		}
	}

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

	for connName := range newMySQLConnections {
		mysqlCfg, ok := newCfg.MySQLConfigFor(connName)
		if !ok {
			return ReloadResult{
				Success: false,
				Error:   fmt.Sprintf("未找到 MySQL 连接 %s", connName),
				Message: "热更新失败",
			}
		}

		if client, exists := s.mysql[connName]; exists {
			var oldMySQL config.MySQLConfig
			var hasOld bool
			if oldCfg != nil {
				oldMySQL, hasOld = oldCfg.MySQLConfigFor(connName)
			}
			if !hasOld || !mysqlConfigEqual(oldMySQL, mysqlCfg) {
				log.Printf("检测到 MySQL 连接 %s 配置变更，准备重建连接", connName)
				_ = client.Close()
				delete(s.mysql, connName)
				exists = false
			}
		}

		if _, exists := s.mysql[connName]; !exists {
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

	for connName := range newRedisConnections {
		redisCfg, ok := newCfg.RedisConfigFor(connName)
		if !ok {
			return ReloadResult{
				Success: false,
				Error:   fmt.Sprintf("未找到 Redis 连接 %s", connName),
				Message: "热更新失败",
			}
		}

		if client, exists := s.redis[connName]; exists {
			var oldRedis config.RedisConfig
			var hasOld bool
			if oldCfg != nil {
				oldRedis, hasOld = oldCfg.RedisConfigFor(connName)
			}
			if !hasOld || !redisConfigEqual(oldRedis, redisCfg) {
				log.Printf("检测到 Redis 连接 %s 配置变更，准备重建连接", connName)
				_ = client.Close()
				delete(s.redis, connName)
				exists = false
			}
		}

		if _, exists := s.redis[connName]; !exists {
			client, err := datasource.NewRedisClient(redisCfg)
			if err != nil {
				return ReloadResult{
					Success: false,
					Error:   fmt.Sprintf("初始化 Redis 连接 %s 失败: %v", connName, err),
					Message: "热更新失败",
				}
			}
			s.redis[connName] = client
		}
	}

	for connName := range newRestAPIConnections {
		restapiCfg, ok := newCfg.RestAPIConfigFor(connName)
		if !ok {
			return ReloadResult{
				Success: false,
				Error:   fmt.Sprintf("未找到 RestAPI 连接 %s", connName),
				Message: "热更新失败",
			}
		}

		if client, exists := s.restapi[connName]; exists {
			var oldRestAPI config.RestAPIConfig
			var hasOld bool
			if oldCfg != nil {
				oldRestAPI, hasOld = oldCfg.RestAPIConfigFor(connName)
			}
			if !hasOld || !restapiConfigEqual(oldRestAPI, restapiCfg) {
				log.Printf("检测到 RestAPI 连接 %s 配置变更，准备重建连接", connName)
				_ = client.Close()
				delete(s.restapi, connName)
				exists = false
			}
		}

		if _, exists := s.restapi[connName]; !exists {
			client, err := datasource.NewRestAPIClient(restapiCfg)
			if err != nil {
				return ReloadResult{
					Success: false,
					Error:   fmt.Sprintf("初始化 RestAPI 连接 %s 失败: %v", connName, err),
					Message: "热更新失败",
				}
			}
			s.restapi[connName] = client
		}
	}

	var newMetrics []string
	var updatedMetrics []metricHolder

	for _, spec := range newCfg.Metrics {
		metricType := spec.Type
		if metricType == "" {
			metricType = "gauge"
		}

		var existingHolder *metricHolder
		for i, holder := range s.metrics {
			if holder.spec.Name == spec.Name {
				existingHolder = &s.metrics[i]
				break
			}
		}

		if existingHolder != nil {
			// 处理 enabled 状态变更
			if (existingHolder.spec.Enabled == nil || *existingHolder.spec.Enabled) && spec.Enabled != nil && !*spec.Enabled {
				// 禁用指标: 从 Prometheus 注销
				s.registry.Unregister(existingHolder.gauge)
				prometheus.DefaultRegisterer.Unregister(existingHolder.gauge)
				existingHolder.spec = spec
				continue
			}
			if existingHolder.spec.Enabled != nil && !*existingHolder.spec.Enabled && (spec.Enabled == nil || *spec.Enabled) {
				// 启用指标: 注册到 Prometheus
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
					var alreadyErr prometheus.AlreadyRegisteredError
					if errors.As(err, &alreadyErr) {
						s.registry.Unregister(alreadyErr.ExistingCollector)
						if retryErr := s.registry.Register(metric); retryErr != nil {
							log.Printf("启用指标 %s 注册失败（重试后）: %v", spec.Name, retryErr)
							continue
						}
					} else {
						log.Printf("启用指标 %s 注册失败: %v", spec.Name, err)
						continue
					}
				}
				if gauge, ok := metric.(prometheus.Gauge); ok {
					existingHolder.gauge = gauge
					existingHolder.spec = spec
					prometheus.DefaultRegisterer.MustRegister(gauge)
				}
				updatedMetrics = append(updatedMetrics, *existingHolder)
				newMetrics = append(newMetrics, spec.Name)
				continue
			}
			if existingHolder.spec.Type != spec.Type || existingHolder.spec.Help != spec.Help || !labelsEqual(existingHolder.spec.Labels, spec.Labels) {
				// 从两个注册表清理旧 metric
				s.registry.Unregister(existingHolder.gauge)
				prometheus.DefaultRegisterer.Unregister(existingHolder.gauge)

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

				// 注册到自定义注册表，处理 AlreadyRegisteredError
				if err := s.registry.Register(metric); err != nil {
					var alreadyErr prometheus.AlreadyRegisteredError
					if errors.As(err, &alreadyErr) {
						s.registry.Unregister(alreadyErr.ExistingCollector)
						if retryErr := s.registry.Register(metric); retryErr != nil {
							return ReloadResult{
								Success: false,
								Error:   fmt.Sprintf("注册指标 %s 失败（重试后）: %v", spec.Name, retryErr),
								Message: "热更新失败",
							}
						}
					} else {
						return ReloadResult{
							Success: false,
							Error:   fmt.Sprintf("注册指标 %s 失败: %v", spec.Name, err),
							Message: "热更新失败",
						}
					}
				}

				if gauge, ok := metric.(prometheus.Gauge); ok {
					existingHolder.gauge = gauge
					existingHolder.spec = spec
					prometheus.DefaultRegisterer.Unregister(existingHolder.gauge)
					prometheus.DefaultRegisterer.MustRegister(gauge)
				}
			} else {
				existingHolder.spec = spec
			}
			updatedMetrics = append(updatedMetrics, *existingHolder)
		} else {
			var metric prometheus.Collector
				if spec.Enabled != nil && !*spec.Enabled {
				continue
		}
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
				prometheus.DefaultRegisterer.MustRegister(gauge)
				newMetrics = append(newMetrics, spec.Name)
			}
		}
	}

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

func mysqlConfigEqual(a, b config.MySQLConfig) bool {
	return a.Host == b.Host &&
		a.Port == b.Port &&
		a.User == b.User &&
		a.Password == b.Password &&
		a.Database == b.Database &&
		reflect.DeepEqual(a.Params, b.Params)
}

func redisConfigEqual(a, b config.RedisConfig) bool {
	return a.Mode == b.Mode &&
		a.Addr == b.Addr &&
		a.Username == b.Username &&
		a.Password == b.Password &&
		a.DB == b.DB &&
		a.EnableTLS == b.EnableTLS &&
		a.SkipTLSVerify == b.SkipTLSVerify
}

func restapiConfigEqual(a, b config.RestAPIConfig) bool {
	return a.BaseURL == b.BaseURL &&
		a.Timeout == b.Timeout &&
		a.TLS.SkipVerify == b.TLS.SkipVerify &&
		a.Retry.MaxAttempts == b.Retry.MaxAttempts &&
		a.Retry.Backoff == b.Retry.Backoff
}
