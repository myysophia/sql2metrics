package datasource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/company/ems-devices/internal/config"
)

// HTTPAPIClient 负责从 HTTP API 获取 JSON 数据并提取指标值。
type HTTPAPIClient struct {
	config config.HTTPAPIConfig
	client *http.Client
}

// NewHTTPAPIClient 创建 HTTP API 客户端。
func NewHTTPAPIClient(cfg config.HTTPAPIConfig) (*HTTPAPIClient, error) {
	if cfg.URL == "" {
		return nil, errors.New("HTTP API 配置缺少 URL")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 // 默认 10 秒
	}

	method := strings.ToUpper(cfg.Method)
	if method == "" {
		method = "GET"
	}

	return &HTTPAPIClient{
		config: cfg,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

// QueryScalar 执行 HTTP 请求并从 JSON 响应中提取指定路径的值。
// jsonPath 支持点号分隔的嵌套路径，如 "main.mqttAuthUrl"
// url 是可选的，如果为空则使用连接配置中的 URL
func (c *HTTPAPIClient) QueryScalar(ctx context.Context, jsonPath string, url ...string) (float64, error) {
	if jsonPath == "" {
		return 0, errors.New("JSON 路径不能为空")
	}

	// 确定使用的 URL：优先使用传入的 url，否则使用配置中的 URL
	targetURL := c.config.URL
	if len(url) > 0 && url[0] != "" {
		targetURL = url[0]
	}
	if targetURL == "" {
		return 0, errors.New("URL 不能为空")
	}

	// 创建 HTTP 请求
	method := strings.ToUpper(c.config.Method)
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		return 0, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置请求头
	if c.config.Headers != nil {
		for k, v := range c.config.Headers {
			req.Header.Set(k, v)
		}
	}

	// 执行请求
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HTTP 请求失败，状态码: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取 HTTP 响应失败: %w", err)
	}

	// 解析 JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("解析 JSON 响应失败: %w", err)
	}

	// 提取指定路径的值
	value, err := extractJSONPath(data, jsonPath)
	if err != nil {
		return 0, fmt.Errorf("提取 JSON 路径 %s 失败: %w", jsonPath, err)
	}

	// 转换为 float64
	return httpValueToFloat(value)
}

// extractJSONPath 从 JSON 数据中提取指定路径的值。
// 支持点号分隔的嵌套路径，如 "main.mqttAuthUrl"
func extractJSONPath(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if part == "" {
			continue
		}

		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("路径 %s 中找不到键 %s", path, part)
			}
			current = val
		case map[interface{}]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("路径 %s 中找不到键 %s", path, part)
			}
			current = val
		default:
			return nil, fmt.Errorf("路径 %s 在 %s 处不是对象类型", path, part)
		}
	}

	return current, nil
}

// httpValueToFloat 将值转换为 float64（HTTP API 专用）。
func httpValueToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		// 尝试解析字符串为数字
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("无法将字符串 %q 转换为数字: %w", v, err)
		}
		return f, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	case nil:
		return 0, errors.New("值为 nil")
	default:
		return 0, fmt.Errorf("不支持的类型: %T", v)
	}
}

// Close 关闭客户端（HTTP 客户端无需关闭，但为了接口一致性保留此方法）。
func (c *HTTPAPIClient) Close() error {
	// HTTP 客户端无需显式关闭
	return nil
}
