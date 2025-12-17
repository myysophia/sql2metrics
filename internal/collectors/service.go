package collectors

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"reflect"
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
	redis      map[string]*datasource.RedisClient
	iotdb      *datasource.IoTDBClient
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
		redis:    make(map[string]*datasource.RedisClient),
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

	// 记录已注册的指标 Help 信息，确保同名指标 Help 一致
	metricHelp := make(map[string]string)
	// 记录已注册的指标唯一标识 (Name + Labels)，避免重复注册导致 panic
	registeredMetrics := make(map[string]bool)

	for _, spec := range cfg.Metrics {
		// 生成唯一标识 key
		labelKey := spec.Name + labelMapToString(spec.Labels)
		if registeredMetrics[labelKey] {
			log.Printf("警告: 指标 %s (Labels: %v) 已注册，跳过重复定义", spec.Name, spec.Labels)
			continue
		}
		registeredMetrics[labelKey] = true

		// 规范化 Help 字符串
		if help, exists := metricHelp[spec.Name]; exists {
			if spec.Help != help {
				log.Printf("警告: 指标 %s 的 Help 字符串不一致 (%q vs %q)，将使用第一个定义的 Help", spec.Name, spec.Help, help)
				spec.Help = help
			}
		} else {
			metricHelp[spec.Name] = spec.Help
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
			prometheus.Unregister(holder.gauge)
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

	newMySQLConnections := mysqlConnectionsNeeded(newCfg)
	newRedisConnections := redisConnectionsNeeded(newCfg)

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

	// 先注销所有旧指标
	for _, holder := range s.metrics {
		s.registry.Unregister(holder.gauge)
	}
	s.metrics = make([]metricHolder, 0)
	
	// 记录已注册的指标 Help 信息，确保同名指标 Help 一致
	metricHelp := make(map[string]string)
	// 记录已注册的指标唯一标识
	registeredMetrics := make(map[string]bool)

	// 用于存储新的指标列表
	var updatedMetrics []metricHolder
	var newMetrics []string

	for _, spec := range newCfg.Metrics {
		labelKey := spec.Name + labelMapToString(spec.Labels)
		if registeredMetrics[labelKey] {
			continue
		}
		registeredMetrics[labelKey] = true

		// 规范化 Help 字符串
		if help, exists := metricHelp[spec.Name]; exists {
			spec.Help = help
		} else {
			metricHelp[spec.Name] = spec.Help
		}

		metricType := spec.Type
		if metricType == "" {
			metricType = "gauge"
		}

		// 新增指标
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
			// 注意：prometheus.MustRegister(gauge) 这里不应该调用 global register，因为我们用的是 s.registry
			newMetrics = append(newMetrics, spec.Name)
		}
	}

	s.metrics = updatedMetrics
	s.cfg = newCfg

	var metricNames []string
	for _, m := range newMetrics {
		metricNames = append(metricNames, m)
	}

	s.metrics = updatedMetrics

	log.Printf("热更新完成: 注册了 %d 个新指标, 总计 %d 个指标", len(newMetrics), len(s.metrics))
	if len(newMetrics) > 0 {
		log.Printf("新注册指标: %v", newMetrics)
	}

	return ReloadResult{
		Success: true,
		Message: fmt.Sprintf("热更新成功 (新增 %d 个指标)", len(newMetrics)),
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
