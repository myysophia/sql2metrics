package datasource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/apache/iotdb-client-go/client"

	"github.com/company/ems-devices/internal/config"
)

// IoTDBClient 负责与 IoTDB 交互获取聚合结果。
type IoTDBClient struct {
	session *client.Session
}

// NewIoTDBClient 初始化 IoTDB 会话。
func NewIoTDBClient(cfg config.IoTDBConfig) (*IoTDBClient, error) {
	if cfg.EnableTLS {
		return nil, errors.New("当前 MVP 暂未支持 IoTDB TLS 连接，请关闭 enable_tls")
	}
	if cfg.Host == "" || cfg.User == "" {
		return nil, errors.New("IoTDB 配置缺少必要字段")
	}
	port := cfg.Port
	if port == 0 {
		port = 6667
	}
	conf := &client.Config{
		Host:     cfg.Host,
		Port:     strconv.Itoa(port),
		UserName: cfg.User,
		Password: cfg.Password,
		FetchSize: func() int32 {
			if cfg.FetchSize <= 0 {
				return client.DefaultFetchSize
			}
			return int32(cfg.FetchSize)
		}(),
		TimeZone: cfg.ZoneID,
	}
	sess := client.NewSession(conf)
	session := &sess
	// 设置连接超时为 5 秒，避免启动时长时间阻塞
	timeout := 5000 // 5 秒超时（毫秒）
	if err := session.Open(cfg.EnableZstd, timeout); err != nil {
		return nil, fmt.Errorf("打开 IoTDB 会话失败: %w", err)
	}
	return &IoTDBClient{session: session}, nil
}

// TestConnection 测试 IoTDB 连接，使用 show databases 命令。
func (c *IoTDBClient) TestConnection(ctx context.Context) error {
	if c.session == nil {
		return errors.New("IoTDB 会话未初始化")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	// 使用 show databases 来测试连接
	dataSet, err := c.session.ExecuteQueryStatement("show databases", nil)
	if err != nil {
		return fmt.Errorf("执行 IoTDB 查询失败: %w", err)
	}
	if dataSet != nil {
		defer dataSet.Close()
	}
	return nil
}

// QueryScalar 执行查询并解析单值结果。
func (c *IoTDBClient) QueryScalar(ctx context.Context, sqlStmt, resultField string) (float64, error) {
	// IoTDB Session 当前不支持 context 取消，此处仅用于对齐接口。
	if c.session == nil {
		return 0, errors.New("IoTDB 会话未初始化")
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	dataSet, err := c.session.ExecuteQueryStatement(sqlStmt, nil)
	if err != nil {
		return 0, fmt.Errorf("执行 IoTDB 查询失败: %w", err)
	}
	if dataSet == nil {
		return 0, errors.New("IoTDB 返回空数据集")
	}
	defer dataSet.Close()

	columns := dataSet.GetColumnNames()
	if len(columns) == 0 {
		return 0, errors.New("IoTDB 结果缺少字段信息")
	}

	target, fallback := pickTargetColumn(columns, resultField)
	if fallback && resultField != "" {
		log.Printf("指定字段 %s 未在 IoTDB 结果中找到，改用列 %s", resultField, target)
	}

	var total float64
	var rows int
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		hasNext, err := dataSet.Next()
		if err != nil {
			return 0, fmt.Errorf("读取 IoTDB 结果失败: %w", err)
		}
		if !hasNext {
			break
		}
		value := dataSet.GetValue(target)
		floatVal, convErr := valueToFloat(target, value)
		if convErr != nil {
			return 0, convErr
		}
		total += floatVal
		rows++
	}
	if rows == 0 {
		return 0, errors.New("IoTDB 查询无结果")
	}
	return total, nil
}

// Close 关闭会话。
func (c *IoTDBClient) Close() error {
	if c.session == nil {
		return nil
	}
	if _, err := c.session.Close(); err != nil {
		return err
	}
	c.session = nil
	return nil
}

func pickTargetColumn(columns []string, hint string) (string, bool) {
	if hint != "" {
		for _, col := range columns {
			if strings.EqualFold(col, hint) {
				return col, false
			}
		}
		lowerHint := strings.ToLower(hint)
		for _, col := range columns {
			if strings.Contains(strings.ToLower(col), lowerHint) {
				return col, true
			}
		}
	}
	return columns[0], hint != ""
}

func valueToFloat(column string, value interface{}) (float64, error) {
	switch v := value.(type) {
	case nil:
		return 0, nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("IoTDB 字段 %s 结果无法解析: %w", column, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("IoTDB 字段 %s 返回不支持的类型: %T", column, v)
	}
}
