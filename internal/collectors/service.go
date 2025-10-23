package collectors

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/company/ems-devices/internal/config"
	"github.com/company/ems-devices/internal/datasource"
)

// Service 负责调度查询并更新 Prometheus 指标。
type Service struct {
	cfg        *config.Config
	mysql      map[string]*datasource.MySQLClient
	iotdb      *datasource.IoTDBClient
	metrics    []metricHolder
	errorCount prometheus.Counter
	lastRun    prometheus.Gauge
	mu         sync.RWMutex
}

type metricHolder struct {
	spec  config.MetricSpec
	gauge prometheus.Gauge
}

// NewService 构造采集服务，按需初始化数据源。
func NewService(cfg *config.Config) (*Service, error) {
	svc := &Service{
		cfg:   cfg,
		mysql: make(map[string]*datasource.MySQLClient),
	}
	var err error
	if needsSource(cfg.Metrics, "iotdb") {
		svc.iotdb, err = datasource.NewIoTDBClient(cfg.IoTDB)
		if err != nil {
			return nil, err
		}
	}

	for connName := range mysqlConnectionsNeeded(cfg) {
		mysqlCfg, ok := cfg.MySQLConfigFor(connName)
		if !ok {
			return nil, fmt.Errorf("未找到 MySQL 连接 %s", connName)
		}
		client, err := datasource.NewMySQLClient(mysqlCfg)
		if err != nil {
			return nil, err
		}
		svc.mysql[connName] = client
	}

	for _, spec := range cfg.Metrics {
		gauge := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        spec.Name,
			Help:        spec.Help,
			ConstLabels: spec.Labels,
		})
		prometheus.MustRegister(gauge)
		svc.metrics = append(svc.metrics, metricHolder{
			spec:  spec,
			gauge: gauge,
		})
	}

	svc.errorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "collector_errors_total",
		Help: "采集周期内出现错误的次数",
	})
	svc.lastRun = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "collector_last_success_timestamp_seconds",
		Help: "最近一次成功采集的 Unix 时间戳",
	})
	prometheus.MustRegister(svc.errorCount, svc.lastRun)
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
}
