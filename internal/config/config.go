package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 描述采集服务的整体配置。
type Config struct {
	Schedule         ScheduleConfig         `yaml:"schedule"`
	Prometheus       PrometheusConfig       `yaml:"prometheus"`
	MySQL            MySQLConfig            `yaml:"mysql"`
	MySQLConnections map[string]MySQLConfig `yaml:"mysql_connections"`
	IoTDB            IoTDBConfig            `yaml:"iotdb"`
	Metrics          []MetricSpec           `yaml:"metrics"`
}

// ScheduleConfig 控制采集周期。
type ScheduleConfig struct {
	Interval string `yaml:"interval"`
}

// PrometheusConfig 定义暴露指标的方式。
type PrometheusConfig struct {
	ListenAddress string `yaml:"listen_address"`
	ListenPort    int    `yaml:"listen_port"`
}

// MySQLConfig 填写 MySQL 连接与查询所需信息。
type MySQLConfig struct {
	Host     string            `yaml:"host"`
	Port     int               `yaml:"port"`
	User     string            `yaml:"user"`
	Password string            `yaml:"password"`
	Database string            `yaml:"database"`
	Params   map[string]string `yaml:"params"`
}

// IoTDBConfig 填写 IoTDB Session 连接信息。
type IoTDBConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	FetchSize   int    `yaml:"fetch_size"`
	ZoneID      string `yaml:"zone_id"`
	EnableTLS   bool   `yaml:"enable_tls"`
	EnableZstd  bool   `yaml:"enable_zstd"`
	SessionPool int    `yaml:"session_pool"`
}

// MetricSpec 定义单个指标查询的元数据。
type MetricSpec struct {
	Name        string            `yaml:"name"`
	Help        string            `yaml:"help"`
	Source      string            `yaml:"source"`
	Query       string            `yaml:"query"`
	Labels      map[string]string `yaml:"labels"`
	ResultField string            `yaml:"result_field"`
	Connection  string            `yaml:"connection"`
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

	if err := cfg.applyDefaults(); err != nil {
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
	for _, m := range c.Metrics {
		if m.Name == "" {
			return errors.New("指标名称不能为空")
		}
		if m.Source != "mysql" && m.Source != "iotdb" {
			return fmt.Errorf("指标 %s 的 source 非法: %s", m.Name, m.Source)
		}
		if m.Query == "" {
			return fmt.Errorf("指标 %s 缺少查询语句", m.Name)
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
	}
	return nil
}

func (c *Config) applyDefaults() error {
	if c.Schedule.Interval == "" {
		c.Schedule.Interval = "1h"
	}
	if c.Prometheus.ListenPort == 0 {
		c.Prometheus.ListenPort = 8080
	}
	if c.MySQLConnections == nil {
		c.MySQLConnections = make(map[string]MySQLConfig)
	}
	if _, ok := c.MySQLConnections["default"]; !ok {
		if c.MySQL.Host != "" || c.MySQL.User != "" || c.MySQL.Database != "" {
			c.MySQLConnections["default"] = c.MySQL
		}
	}
	if c.IoTDB.FetchSize == 0 {
		c.IoTDB.FetchSize = 1024
	}
	if c.IoTDB.ZoneID == "" {
		c.IoTDB.ZoneID = "UTC+08:00"
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
