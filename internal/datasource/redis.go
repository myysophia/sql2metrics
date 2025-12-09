package datasource

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/company/ems-devices/internal/config"
)

// RedisClient 封装 Redis 只读查询能力。
type RedisClient struct {
	client redis.UniversalClient
	mode   string
}

// NewRedisClient 基于配置创建 Redis 客户端。
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	mode := cfg.Mode
	if mode == "" {
		mode = "standalone"
	}
	if cfg.Addr == "" {
		return nil, errors.New("Redis 配置缺少 addr")
	}
	if mode != "standalone" {
		return nil, fmt.Errorf("当前仅支持 standalone 模式，收到: %s", mode)
	}

	opt := &redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.EnableTLS {
		opt.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify,
		}
	}

	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("Redis 连接验证失败: %w", err)
	}

	return &RedisClient{
		client: client,
		mode:   mode,
	}, nil
}

// QueryScalar 执行只读命令并解析为浮点结果。
func (c *RedisClient) QueryScalar(ctx context.Context, raw string) (float64, error) {
	if c.client == nil {
		return 0, errors.New("Redis 客户端未初始化")
	}
	cmd, args, err := parseRedisCommand(raw)
	if err != nil {
		return 0, err
	}

	params := make([]interface{}, 0, len(args)+1)
	params = append(params, cmd)
	for _, a := range args {
		params = append(params, a)
	}

	result, err := c.client.Do(ctx, params...).Result()
	if errors.Is(err, redis.Nil) {
		return 0, fmt.Errorf("Redis 命令 %s 未返回结果", cmd)
	}
	if err != nil {
		return 0, fmt.Errorf("执行 Redis 命令失败: %w", err)
	}
	return redisValueToFloat(result)
}

// Ping 测试连接。
func (c *RedisClient) Ping(ctx context.Context) error {
	if c.client == nil {
		return errors.New("Redis 客户端未初始化")
	}
	return c.client.Ping(ctx).Err()
}

// Close 释放连接资源。
func (c *RedisClient) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func parseRedisCommand(raw string) (string, []string, error) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", nil, errors.New("Redis 命令不能为空")
	}
	cmd := strings.ToUpper(fields[0])
	if _, ok := allowedRedisCommands()[cmd]; !ok {
		return "", nil, fmt.Errorf("Redis 命令 %s 不被允许，请使用只读命令", cmd)
	}
	return cmd, fields[1:], nil
}

func allowedRedisCommands() map[string]struct{} {
	return map[string]struct{}{
		"GET":     {},
		"HGET":    {},
		"LLEN":    {},
		"SCARD":   {},
		"ZCARD":   {},
		"PFCOUNT": {},
		"STRLEN":  {},
		"HLEN":    {},
		"ZCOUNT":  {},
		"EXISTS":  {},
		"ZSCORE":  {},
		"DBSIZE":  {},
	}
}

func redisValueToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case nil:
		return 0, errors.New("Redis 命令返回空值")
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("Redis 返回值无法转换为数字: %w", err)
		}
		return f, nil
	case []byte:
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return 0, fmt.Errorf("Redis 返回值无法转换为数字: %w", err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("Redis 返回不支持的类型: %T", v)
	}
}
