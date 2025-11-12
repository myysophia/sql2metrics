package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/company/ems-devices/internal/config"
)

// MySQLClient 封装 MySQL 查询能力。
type MySQLClient struct {
	db *sql.DB
}

// NewMySQLClient 基于配置创建连接池。
func NewMySQLClient(cfg config.MySQLConfig) (*MySQLClient, error) {
	dsn, err := cfg.DSN()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("初始化 MySQL 连接失败: %w", err)
	}
	// 采用保守连接池设置，避免穿透数据库。
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(2)
	db.SetMaxOpenConns(5)
	
	// 设置连接超时上下文，避免启动时长时间阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("MySQL 连接验证失败: %w", err)
	}
	return &MySQLClient{db: db}, nil
}

// QueryScalar 执行聚合查询，返回单一数值结果。
func (c *MySQLClient) QueryScalar(ctx context.Context, sqlStmt string) (float64, error) {
	var value sql.NullFloat64
	if err := c.db.QueryRowContext(ctx, sqlStmt).Scan(&value); err != nil {
		return 0, fmt.Errorf("执行 MySQL 查询失败: %w", err)
	}
	if !value.Valid {
		return 0, fmt.Errorf("MySQL 查询未返回有效结果")
	}
	return value.Float64, nil
}

// Close 收回底层资源。
func (c *MySQLClient) Close() error {
	return c.db.Close()
}

// Ping 测试数据库连接。
func (c *MySQLClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
