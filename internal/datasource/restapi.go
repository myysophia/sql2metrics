package datasource

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/company/ems-devices/internal/config"
)

// RestAPIClient 封装 RESTful API 查询能力。
type RestAPIClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	retry   config.RestAPIRetryConfig
}

// NewRestAPIClient 基于配置创建 REST API 客户端。
func NewRestAPIClient(cfg config.RestAPIConfig) (*RestAPIClient, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("RestAPI 配置缺少 base_url")
	}

	// 解析超时时间
	timeout := 30 * time.Second
	if cfg.Timeout != "" {
		parsed, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("解析 RestAPI 超时配置失败: %w", err)
		}
		timeout = parsed
	}

	// 配置 TLS - 始终设置 TLS 配置以确保兼容性
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// 显式添加 RSA 密码套件，用于支持不使用 ECDHE 的服务器
		// 某些服务器（如 control.pingjl.com）只支持纯 RSA 密码套件
		CipherSuites: []uint16{
			// ECDHE 密码套件（优先）
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			// RSA 密码套件（用于兼容老旧服务器）
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// 尝试解析基础 URL 以设置 SNI (ServerName)
	if u, err := url.Parse(cfg.BaseURL); err == nil {
		tlsConfig.ServerName = u.Hostname()
	}

	if cfg.TLS.SkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		DisableKeepAlives:   false,
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second,
		// 对于某些老旧服务器或特殊代理，显式禁用 HTTP2 可能更有助于握手成功
		ForceAttemptHTTP2: false,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	// 标准化 baseURL（移除末尾斜杠）
	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	return &RestAPIClient{
		client:  client,
		baseURL: baseURL,
		headers: cfg.Headers,
		retry:   cfg.Retry,
	}, nil
}

// QueryScalar 执行 HTTP 请求并从 JSON 响应中提取数值。
// query 格式支持：
//   - "GET /path"
//   - "POST /path\n{json_body}"
func (c *RestAPIClient) QueryScalar(ctx context.Context, query, resultField string) (float64, error) {
	method, path, body, err := parseQuery(query)
	if err != nil {
		return 0, err
	}

	url := c.baseURL + path

	// 执行请求（带重试）
	maxAttempts := 1
	if c.retry.MaxAttempts > 0 {
		maxAttempts = c.retry.MaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := c.doRequest(ctx, method, url, body)
		if err == nil {
			return extractJSONValue(result, resultField)
		}
		lastErr = err

		// 最后一次尝试不需要等待
		if attempt < maxAttempts {
			backoff := time.Second
			if c.retry.Backoff != "" {
				if parsed, parseErr := time.ParseDuration(c.retry.Backoff); parseErr == nil {
					backoff = parsed
				}
			}
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return 0, fmt.Errorf("RestAPI 请求失败（重试 %d 次）: %w", maxAttempts, lastErr)
}

// QueryRaw 执行 HTTP 请求并返回完整的 JSON 响应，用于预览和字段选择。
func (c *RestAPIClient) QueryRaw(ctx context.Context, query string) (interface{}, error) {
	method, path, body, err := parseQuery(query)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + path
	return c.doRequest(ctx, method, url, body)
}

// doRequest 执行单次 HTTP 请求。
func (c *RestAPIClient) doRequest(ctx context.Context, method, url string, body string) (interface{}, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	// 设置默认 Content-Type
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置自定义请求头（跳过空值）
	for key, value := range c.headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行 HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP 请求返回非成功状态码 %d: %s", resp.StatusCode, string(bodyBytes))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 JSON 响应失败: %w", err)
	}

	return result, nil
}

// Ping 测试 API 连通性（发送 HEAD 或 GET 请求到 base_url）。
func (c *RestAPIClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL, nil)
	if err != nil {
		return fmt.Errorf("创建测试请求失败: %w", err)
	}

	for key, value := range c.headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("RestAPI 连接测试失败: %w", err)
	}
	defer resp.Body.Close()

	// 只要能收到响应就认为连接正常（忽略具体状态码）
	return nil
}

// Close 释放资源（HTTP 客户端不需要显式关闭）。
func (c *RestAPIClient) Close() error {
	return nil
}

// parseQuery 解析查询字符串，返回 HTTP 方法、路径和请求体。
func parseQuery(query string) (method, path, body string, err error) {
	query = strings.TrimSpace(query)
	// 允许空查询，默认使用 GET 方法直接请求 baseURL
	if query == "" {
		return "GET", "", "", nil
	}

	// 按换行符分割，第一行是 "METHOD /path"，后续行是 body
	lines := strings.SplitN(query, "\n", 2)
	firstLine := strings.TrimSpace(lines[0])

	parts := strings.SplitN(firstLine, " ", 2)
	// 支持两种格式：
	// 1. "GET" - 仅方法，路径为空（直接请求 baseURL）
	// 2. "GET /path" - 方法 + 路径
	if len(parts) == 1 {
		// 仅方法名，无路径
		method = strings.ToUpper(strings.TrimSpace(parts[0]))
		path = ""
	} else {
		method = strings.ToUpper(strings.TrimSpace(parts[0]))
		path = strings.TrimSpace(parts[1])
	}

	// 验证 HTTP 方法
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}
	if !validMethods[method] {
		return "", "", "", fmt.Errorf("不支持的 HTTP 方法: %s", method)
	}

	// 如果有路径且不以 / 开头，添加 /
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 提取 body
	if len(lines) > 1 {
		body = strings.TrimSpace(lines[1])
	}

	return method, path, body, nil
}

// extractJSONValue 从 JSON 数据中根据路径提取数值。
// 支持的路径格式：
//   - "data.count" - 嵌套对象
//   - "items[0].value" - 数组索引
//   - "length" - 特殊关键字，返回数组长度
func extractJSONValue(data interface{}, path string) (float64, error) {
	if path == "" {
		// 如果没有指定路径，尝试直接转换
		return toFloat(data)
	}

	// 特殊处理 "length" 关键字
	if path == "length" {
		if arr, ok := data.([]interface{}); ok {
			return float64(len(arr)), nil
		}
		return 0, errors.New("'length' 只能用于数组类型")
	}

	current := data
	parts := splitPath(path)

	for _, part := range parts {
		if current == nil {
			return 0, fmt.Errorf("路径 %s 中遇到 nil 值", path)
		}

		// 检查是否是数组索引访问
		if idx, isIndex := parseArrayIndex(part); isIndex {
			arr, ok := current.([]interface{})
			if !ok {
				return 0, fmt.Errorf("路径 %s: 期望数组类型，实际为 %T", part, current)
			}
			if idx < 0 || idx >= len(arr) {
				return 0, fmt.Errorf("路径 %s: 数组索引 %d 越界（长度 %d）", part, idx, len(arr))
			}
			current = arr[idx]
		} else {
			// 对象属性访问
			obj, ok := current.(map[string]interface{})
			if !ok {
				return 0, fmt.Errorf("路径 %s: 期望对象类型，实际为 %T", part, current)
			}
			val, exists := obj[part]
			if !exists {
				return 0, fmt.Errorf("路径 %s: 字段 %s 不存在", path, part)
			}
			current = val
		}
	}

	return toFloat(current)
}

// splitPath 分割路径字符串。
// 例如 "data.items[0].value" -> ["data", "items", "[0]", "value"]
func splitPath(path string) []string {
	var parts []string
	current := ""

	for i := 0; i < len(path); i++ {
		ch := path[i]
		switch ch {
		case '.':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		case '[':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			// 找到匹配的 ]
			j := i + 1
			for j < len(path) && path[j] != ']' {
				j++
			}
			if j < len(path) {
				parts = append(parts, path[i:j+1])
				i = j
			}
		default:
			current += string(ch)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// parseArrayIndex 解析数组索引，例如 "[0]" -> 0, true
func parseArrayIndex(part string) (int, bool) {
	if !strings.HasPrefix(part, "[") || !strings.HasSuffix(part, "]") {
		return 0, false
	}
	indexStr := part[1 : len(part)-1]
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		return 0, false
	}
	return idx, true
}

// toFloat 将各种类型转换为 float64。
func toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case nil:
		return 0, errors.New("值为 nil")
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("字符串 %q 无法转换为数字: %w", v, err)
		}
		return parsed, nil
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf("不支持的类型 %T 转换为数字", v)
	}
}
