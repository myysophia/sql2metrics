package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	raw := `
schedule:
  interval: 30m
prometheus:
  listen_address: 127.0.0.1
mysql:
  host: localhost
  user: tester
  password: secret
  database: nova_energy
metrics:
  - name: sample_total
    help: 样例指标
    source: mysql
    query: SELECT 1
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("载入配置失败: %v", err)
	}

	if cfg.Prometheus.ListenPort != 8080 {
		t.Fatalf("Prometheus 默认端口期望 8080，实际 %d", cfg.Prometheus.ListenPort)
	}
	if len(cfg.Metrics) != 1 {
		t.Fatalf("应解析出 1 个指标，实际 %d", len(cfg.Metrics))
	}
	if _, err := cfg.Schedule.IntervalDuration(); err != nil {
		t.Fatalf("应成功解析采集周期: %v", err)
	}
}

func TestValidateMultiMySQLConnections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	raw := `
mysql_connections:
  default:
    host: 127.0.0.1
    user: readonly
    password: secret
    database: nova_energy
  business:
    host: 127.0.0.1
    user: readonly
    password: secret
    database: nova_energy_cloud
metrics:
  - name: energy_business_total
    help: 工商业储能设备
    source: mysql
    query: SELECT COUNT(1) FROM station_device
    connection: business
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("载入配置失败: %v", err)
	}

	if _, ok := cfg.MySQLConfigFor("business"); !ok {
		t.Fatalf("应当解析出 business 连接配置")
	}
	businessDSN, err := cfg.MySQLConnections["business"].DSN()
	if err != nil {
		t.Fatalf("生成 DSN 不应失败: %v", err)
	}
	if want := "readonly:secret@tcp(127.0.0.1:3306)/nova_energy_cloud"; businessDSN != want {
		t.Fatalf("business DSN 期望前缀 %s，实际 %s", want, businessDSN)
	}
}

func TestValidateMissingMySQLConnection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	raw := `
metrics:
  - name: energy_business_total
    help: 工商业储能设备
    source: mysql
    query: SELECT COUNT(1) FROM station_device
    connection: business
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("缺少 business 连接时应当返回错误")
	}
}

func TestEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	if err := os.Setenv("TEST_MYSQL_USER", "env_user"); err != nil {
		t.Fatalf("设置环境变量失败: %v", err)
	}
	defer os.Unsetenv("TEST_MYSQL_USER")

	raw := `
mysql:
  host: localhost
  user: ${TEST_MYSQL_USER}
  database: nova_energy
metrics:
  - name: sample_total
    help: 样例指标
    source: mysql
    query: SELECT 1
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("写入配置失败: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("载入配置失败: %v", err)
	}
	mysqlCfg, ok := cfg.MySQLConfigFor("default")
	if !ok {
		t.Fatalf("应当返回 default MySQL 配置")
	}
	if mysqlCfg.User != "env_user" {
		t.Fatalf("期望从环境变量读到用户 env_user，实际 %s", mysqlCfg.User)
	}
}
