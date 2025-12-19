package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// labelNameRegex 匹配有效的 Prometheus label 名称
var labelNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidLabelName 检查 label 名称是否符合 Prometheus 规范
func isValidLabelName(name string) bool {
	return labelNameRegex.MatchString(name)
}

// Config 描述采集服务的整体配置。
type Config struct {
	Schedule            ScheduleConfig            `yaml:"schedule" json:"schedule"`
	Prometheus          PrometheusConfig          `yaml:"prometheus" json:"prometheus"`
	MySQL               MySQLConfig               `yaml:"mysql" json:"mysql"`
	MySQLConnections    map[string]MySQLConfig    `yaml:"mysql_connections" json:"mysql_connections"`
	Redis               RedisConfig               `yaml:"redis" json:"redis"`
	RedisConnections    map[string]RedisConfig    `yaml:"redis_connections" json:"redis_connections"`
	RestAPIConnections  map[string]RestAPIConfig  `yaml:"restapi_connections" json:"restapi_connections"`
	IoTDB               IoTDBConfig               `yaml:"iotdb" json:"iotdb"`
	Metrics             []MetricSpec              `yaml:"metrics" json:"metrics"`
}

// ScheduleConfig 控制采集周期。
type ScheduleConfig struct {
	Interval string `yaml:"interval" json:"interval"`
}

// PrometheusConfig 定义暴露指标的方式。
type PrometheusConfig struct {
	ListenAddress string `yaml:"listen_address" json:"listen_address"`
	ListenPort    int    `yaml:"listen_port" json:"listen_port"`
}

// MySQLConfig 填写 MySQL 连接与查询所需信息。
type MySQLConfig struct {
	Host     string            `yaml:"host" json:"host"`
	Port     int               `yaml:"port" json:"port"`
	User     string            `yaml:"user" json:"user"`
	Password string            `yaml:"password" json:"password"`
	Database string            `yaml:"database" json:"database"`
	Params   map[string]string `yaml:"params" json:"params,omitempty"`
}

// RedisConfig 填写 Redis 连接信息。
type RedisConfig struct {
	Mode          string `yaml:"mode" json:"mode"` // standalone/sentinel/cluster，当前仅支持 standalone
	Addr          string `yaml:"addr" json:"addr"` // host:port
	Username      string `yaml:"username" json:"username,omitempty"`
	Password      string `yaml:"password" json:"password,omitempty"`
	DB            int    `yaml:"db" json:"db,omitempty"`
	EnableTLS     bool   `yaml:"enable_tls" json:"enable_tls,omitempty"`
	SkipTLSVerify bool   `yaml:"skip_tls_verify" json:"skip_tls_verify,omitempty"`
}

// IoTDBConfig 填写 IoTDB Session 连接信息。
type IoTDBConfig struct {
	Host        string `yaml:"host" json:"host"`
	Port        int    `yaml:"port" json:"port"`
	User        string `yaml:"user" json:"user"`
	Password    string `yaml:"password" json:"password"`
	FetchSize   int    `yaml:"fetch_size" json:"fetch_size"`
	ZoneID      string `yaml:"zone_id" json:"zone_id"`
	EnableTLS   bool   `yaml:"enable_tls" json:"enable_tls"`
	EnableZstd  bool   `yaml:"enable_zstd" json:"enable_zstd"`
	SessionPool int    `yaml:"session_pool" json:"session_pool,omitempty"`
}

// RestAPIConfig 填写 RESTful API 连接信息。
type RestAPIConfig struct {
	BaseURL string            `yaml:"base_url" json:"base_url"`
	Timeout string            `yaml:"timeout" json:"timeout,omitempty"`
	Headers map[string]string `yaml:"headers" json:"headers,omitempty"`
	TLS     RestAPITLSConfig  `yaml:"tls" json:"tls,omitempty"`
	Retry   RestAPIRetryConfig `yaml:"retry" json:"retry,omitempty"`
}

// RestAPITLSConfig 定义 RestAPI TLS 配置。
type RestAPITLSConfig struct {
	SkipVerify bool `yaml:"skip_verify" json:"skip_verify,omitempty"`
}

// RestAPIRetryConfig 定义 RestAPI 重试策略。
type RestAPIRetryConfig struct {
	MaxAttempts int    `yaml:"max_attempts" json:"max_attempts,omitempty"`
	Backoff     string `yaml:"backoff" json:"backoff,omitempty"`
}

// MetricSpec 定义单个指标查询的元数据。
type MetricSpec struct {
	Name        string              `yaml:"name" json:"name"`
	Help        string              `yaml:"help" json:"help"`
	Type        string              `yaml:"type" json:"type"` // gauge/counter/histogram/summary，默认为 gauge
	Source      string              `yaml:"source" json:"source"`
	Query       string              `yaml:"query" json:"query"`
	Labels      map[string]string   `yaml:"labels" json:"labels,omitempty"`
	ResultField string              `yaml:"result_field" json:"result_field,omitempty"`
	Connection  string              `yaml:"connection" json:"connection,omitempty"`
	Buckets     []float64           `yaml:"buckets,omitempty" json:"buckets,omitempty"` // Histogram 分桶
	Objectives  map[float64]float64 `yaml:"objectives,omitempty" json:"-"`              // Summary 分位数目标（JSON 序列化通过 ObjectivesJSON）
}

// ObjectivesJSON 用于 JSON 序列化的 objectives（使用字符串 key）。
type ObjectivesJSON map[string]float64

// MarshalJSON 实现自定义 JSON 序列化。
func (m MetricSpec) MarshalJSON() ([]byte, error) {
	type Alias MetricSpec
	aux := &struct {
		Objectives ObjectivesJSON `json:"objectives,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(&m),
	}

	if m.Objectives != nil && len(m.Objectives) > 0 {
		aux.Objectives = make(ObjectivesJSON)
		for k, v := range m.Objectives {
			aux.Objectives[fmt.Sprintf("%g", k)] = v
		}
	}

	return json.Marshal(aux)
}

// UnmarshalJSON 实现自定义 JSON 反序列化。
func (m *MetricSpec) UnmarshalJSON(data []byte) error {
	type Alias MetricSpec
	aux := &struct {
		Objectives ObjectivesJSON `json:"objectives,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Objectives != nil && len(aux.Objectives) > 0 {
		m.Objectives = make(map[float64]float64)
		for k, v := range aux.Objectives {
			key, err := strconv.ParseFloat(k, 64)
			if err != nil {
				return fmt.Errorf("解析 objectives key 失败: %w", err)
			}
			m.Objectives[key] = v
		}
	}

	return nil
}

// Load 读取并解析 YAML 配置文件。
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	expanded := os.ExpandEnv(string(raw))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := cfg.ApplyDefaults(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IntervalDuration 解析计划采集间隔。
func (s ScheduleConfig) IntervalDuration() (time.Duration, error) {
	interval := s.Interval
	if interval == "" {
		interval = "1h"
	}
	d, err := time.ParseDuration(interval)
	if err != nil {
		return 0, fmt.Errorf("解析采集周期失败: %w", err)
	}
	return d, nil
}

// ListenAddr 拼接监听地址。
func (p PrometheusConfig) ListenAddr() string {
	host := p.ListenAddress
	if host == "" {
		host = "0.0.0.0"
	}
	port := p.ListenPort
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// DSN 返回 MySQL DSN。
func (m MySQLConfig) DSN() (string, error) {
	if m.Host == "" || m.User == "" || m.Database == "" {
		return "", errors.New("MySQL 配置缺少必要字段")
	}
	port := m.Port
	if port == 0 {
		port = 3306
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.User, m.Password, m.Host, port, m.Database)
	if len(m.Params) == 0 {
		return dsn, nil
	}
	first := true
	for k, v := range m.Params {
		sep := "?"
		if !first {
			sep = "&"
		}
		dsn += fmt.Sprintf("%s%s=%s", sep, k, v)
		first = false
	}
	return dsn, nil
}

// Validate 检查配置完整性。
func (c *Config) Validate() error {
	if len(c.Metrics) == 0 {
		return errors.New("至少需要定义一个指标")
	}
	if c.MySQLConnections == nil {
		c.MySQLConnections = make(map[string]MySQLConfig)
	}
	if c.RedisConnections == nil {
		c.RedisConnections = make(map[string]RedisConfig)
	}
	for name, rc := range c.RedisConnections {
		if rc.Addr == "" {
			return fmt.Errorf("Redis 连接 %s 缺少 addr", name)
		}
		mode := rc.Mode
		if mode == "" {
			mode = "standalone"
		}
		if mode != "standalone" {
			return fmt.Errorf("Redis 连接 %s 使用的模式暂未支持: %s", name, mode)
		}
	}
	for _, m := range c.Metrics {
		if m.Name == "" {
			return errors.New("指标名称不能为空")
		}
		if m.Source != "mysql" && m.Source != "iotdb" && m.Source != "redis" && m.Source != "restapi" {
			return fmt.Errorf("指标 %s 的 source 非法: %s", m.Name, m.Source)
		}
		// RestAPI 类型允许查询为空（直接请求 base_url）
		if m.Query == "" && m.Source != "restapi" {
			return fmt.Errorf("指标 %s 缺少查询语句", m.Name)
		}
		metricType := m.Type
		if metricType == "" {
			metricType = "gauge"
		}
		if metricType != "gauge" && metricType != "counter" && metricType != "histogram" && metricType != "summary" {
			return fmt.Errorf("指标 %s 的类型非法: %s，支持的类型: gauge, counter, histogram, summary", m.Name, metricType)
		}
		if metricType == "histogram" && len(m.Buckets) == 0 {
			return fmt.Errorf("指标 %s 类型为 histogram，但未配置 buckets", m.Name)
		}
		if metricType == "summary" && len(m.Objectives) == 0 {
			return fmt.Errorf("指标 %s 类型为 summary，但未配置 objectives", m.Name)
		}
		// 验证 label 名称格式（必须以字母或下划线开头，只能包含字母、数字、下划线）
		for labelName := range m.Labels {
			if !isValidLabelName(labelName) {
				return fmt.Errorf("指标 %s 的 label 名称 %q 无效，必须以字母或下划线开头，只能包含字母、数字和下划线", m.Name, labelName)
			}
		}
		if m.Source == "mysql" {
			conn := m.Connection
			if conn == "" {
				conn = "default"
			}
			if _, ok := c.MySQLConnections[conn]; !ok {
				return fmt.Errorf("指标 %s 引用的 MySQL 连接 %s 未配置", m.Name, conn)
			}
		}
		if m.Source == "redis" {
			conn := m.Connection
			if conn == "" {
				conn = "default"
			}
			if _, ok := c.RedisConnections[conn]; !ok {
				return fmt.Errorf("指标 %s 引用的 Redis 连接 %s 未配置", m.Name, conn)
			}
		}
		if m.Source == "restapi" {
			conn := m.Connection
			if conn == "" {
				conn = "default"
			}
			if _, ok := c.RestAPIConnections[conn]; !ok {
				return fmt.Errorf("指标 %s 引用的 RestAPI 连接 %s 未配置", m.Name, conn)
			}
		}
	}
	return nil
}

// ApplyDefaults 应用默认值到配置。
func (c *Config) ApplyDefaults() error {
	if c.Schedule.Interval == "" {
		c.Schedule.Interval = "1h"
	}
	if c.Prometheus.ListenPort == 0 {
		c.Prometheus.ListenPort = 8080
	}
	if c.MySQLConnections == nil {
		c.MySQLConnections = make(map[string]MySQLConfig)
	}
	if c.RedisConnections == nil {
		c.RedisConnections = make(map[string]RedisConfig)
	}
	if _, ok := c.MySQLConnections["default"]; !ok {
		if c.MySQL.Host != "" || c.MySQL.User != "" || c.MySQL.Database != "" {
			c.MySQLConnections["default"] = c.MySQL
		}
	}
	if _, ok := c.RedisConnections["default"]; !ok {
		if c.Redis.Addr != "" {
			c.RedisConnections["default"] = c.Redis
		}
	}
	for name, rc := range c.RedisConnections {
		if rc.Mode == "" {
			rc.Mode = "standalone"
		}
		c.RedisConnections[name] = rc
	}
	if c.IoTDB.FetchSize == 0 {
		c.IoTDB.FetchSize = 1024
	}
	if c.IoTDB.ZoneID == "" {
		c.IoTDB.ZoneID = "UTC+08:00"
	}
	for i := range c.Metrics {
		if c.Metrics[i].Type == "" {
			c.Metrics[i].Type = "gauge"
		}
	}
	if c.RestAPIConnections == nil {
		c.RestAPIConnections = make(map[string]RestAPIConfig)
	}
	return nil
}

// MySQLConfigFor 返回指定名称的 MySQL 配置，默认为 default。
func (c *Config) MySQLConfigFor(name string) (MySQLConfig, bool) {
	if name == "" {
		name = "default"
	}
	if c.MySQLConnections == nil {
		return MySQLConfig{}, false
	}
	conf, ok := c.MySQLConnections[name]
	return conf, ok
}

// RedisConfigFor 返回指定名称的 Redis 配置，默认为 default。
func (c *Config) RedisConfigFor(name string) (RedisConfig, bool) {
	if name == "" {
		name = "default"
	}
	if c.RedisConnections == nil {
		return RedisConfig{}, false
	}
	conf, ok := c.RedisConnections[name]
	return conf, ok
}

// Save 将配置保存到文件。
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// RestAPIConfigFor 返回指定名称的 RestAPI 配置，默认为 default。
func (c *Config) RestAPIConfigFor(name string) (RestAPIConfig, bool) {
	if name == "" {
		name = "default"
	}
	if c.RestAPIConnections == nil {
		return RestAPIConfig{}, false
	}
	conf, ok := c.RestAPIConnections[name]
	return conf, ok
}

// Clone 创建配置的深拷贝
func (c *Config) Clone() *Config {
	// 使用 JSON 序列化/反序列化来实现深拷贝
	data, _ := json.Marshal(c)
	var clone Config
	_ = json.Unmarshal(data, &clone)
	return &clone
}
