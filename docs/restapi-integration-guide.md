# 接入 RESTful API 数据源指南

本指南以阿里云函数计算（FC）调用次数查询 API 为例，演示如何将任意 RESTful API 接入 sql2metrics 并转化为 Prometheus 指标。

## 实际 API 示例

```
GET http://13.236.113.51:9098/api/v1/fc/invocations?function=mineru&minutes=600&region=cn-hangzhou&timezone=Asia/Shanghai
```

响应：
```json
{
  "function": "mineru",
  "region": "cn-hangzhou",
  "timezone": "Asia/Shanghai",
  "period": 60,
  "start_time": "2026-05-07T03:58:28+08:00",
  "end_time": "2026-05-07T13:58:28+08:00",
  "datapoints": 227,
  "data": {
    "value": 2443
  }
}
```

我们关注的是 `data.value`（调用次数：2443）。

---

## 第一步：确认 API 响应结构

先用 `curl` 调用 API，确认返回的是合法 JSON：

```bash
curl -s 'http://13.236.113.51:9098/api/v1/fc/invocations?function=mineru&minutes=600&region=cn-hangzhou&timezone=Asia/Shanghai' | python3 -m json.tool
```

**要求**：
- 返回 `Content-Type: application/json`
- HTTP 状态码 200
- 响应体是合法 JSON 对象或数组

---

## 第二步：编写配置

在配置文件的 `restapi_connections` 和 `metrics` 两个位置添加配置：

```yaml
# 1. 定义 API 连接（连接池管理，多个指标可复用同一个连接）
restapi_connections:
  bbd-mineru:                          # 连接名称，自定义标识
    base_url: http://13.236.113.51:9098/api/v1/fc/invocations?function=mineru&minutes=600&region=cn-hangzhou&timezone=Asia/Shanghai
    timeout: 30s
    headers: {}
    tls:
      skip_verify: false
    retry:
      max_attempts: 0
      backoff: ""

# 2. 定义指标，引用上面的连接
metrics:
  - name: bbd_mineru                     # Prometheus 指标名称（全局唯一）
    help: mineru 调用情况                 # 帮助信息
    type: gauge                          # 指标类型: gauge / counter / histogram / summary
    source: restapi                      # 数据源类型固定为 restapi
    query: ""                            # 空=GET 请求 base_url；也可写 "GET /path" 或 "POST /path\n{json}"
    result_field: data.value             # 从 JSON 响应中提取数值的路径
    connection: bbd-mineru               # 引用 restapi_connections 中的连接名
    labels:                              # Prometheus 标签
      env: prod
      region: ch-hangzhou
    enabled: true                        # 是否启用采集
```

---

## 配置字段说明

### restapi_connections（连接配置）

| 字段 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `base_url` | 是 | API 完整地址，包含查询参数 | `http://host:port/api/v1/resource?type=A` |
| `timeout` | 否 | 请求超时时间，默认 `30s` | `10s`、`60s` |
| `headers` | 否 | 自定义请求头 | `{"Authorization": "Bearer xxx"}` |
| `tls.skip_verify` | 否 | 跳过 TLS 证书验证，默认 `false` | `true`（自签名证书时用） |
| `retry.max_attempts` | 否 | 失败重试次数，默认 `0`（不重试） | `3` |
| `retry.backoff` | 否 | 重试间隔，默认 `1s` | `5s` |

### metrics（指标配置）

| 字段 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `name` | 是 | Prometheus 指标名称，全局唯一 | `bbd_mineru` |
| `source` | 是 | 固定为 `restapi` | `restapi` |
| `connection` | 是 | 引用 `restapi_connections` 中的连接名 | `bbd-mineru` |
| `query` | 否 | HTTP 方法+路径，空=直接 GET `base_url` | `GET /extra/path` |
| `result_field` | 是 | 从 JSON 响应提取数值的路径 | `data.value` |
| `type` | 否 | 指标类型，默认 `gauge` | `gauge` |
| `enabled` | 否 | 是否启用，默认 `true` | `false` |
| `labels` | 否 | Prometheus 标签 | `{"env": "prod"}` |

---

## result_field 路径语法

`result_field` 支持以下格式：

| 语法 | 含义 | 示例 |
|------|------|------|
| `data.value` | 嵌套对象取值 | `{"data": {"value": 100}}` → `100` |
| `data.items[0].count` | 数组索引取值 | `{"data": {"items": [{"count": 5}]}}` → `5` |
| `data.total` | 同上 | `{"data": {"total": 42}}` → `42` |
| `datapoints` | 顶层字段 | `{"datapoints": 10}` → `10` |
| `length` | 特殊关键字，取数组长度 | `{"items": [1,2,3]}` → `3` |

**规则**：用 `.` 分隔层级，用 `[N]` 访问数组第 N 个元素（从 0 开始）。

---

## 不同 query 写法示例

### 1. 直接请求 base_url（最常见）

```yaml
restapi_connections:
  my-api:
    base_url: http://host:port/api/v1/status?app=myapp

metrics:
  - name: my_status
    source: restapi
    query: ""                          # 空=GET base_url
    result_field: data.count
    connection: my-api
```

### 2. 追加路径

```yaml
restapi_connections:
  my-api:
    base_url: http://host:port/api/v1

metrics:
  - name: my_status
    source: restapi
    query: "GET /status?app=myapp"     # 拼接为 http://host:port/api/v1/status?app=myapp
    result_field: data.count
    connection: my-api
```

### 3. POST 请求带 Body

```yaml
metrics:
  - name: my_metric
    source: restapi
    query: |
      POST /query
      {"filter": "region=cn-hangzhou"}
    result_field: result.value
    connection: my-api
```

### 4. 需要认证的 API

```yaml
restapi_connections:
  secure-api:
    base_url: https://api.example.com/v1/metrics
    headers:
      Authorization: "Bearer your-token-here"
    tls:
      skip_verify: false
```

---

## 验证

### 1. 保存并应用配置后检查日志

```bash
journalctl -u sql2metrics-v2.8.service -f | grep bbd_mineru
```

成功日志：
```
开始更新指标 bbd_mineru (source=restapi)
执行 RestAPI 查询（连接=bbd-mineru）:
指标 bbd_mineru 更新成功，值=2443.000，耗时=827ms
```

### 2. 检查 Prometheus 指标端点

```bash
curl -s http://localhost:18888/metrics | grep bbd_mineru
```

输出：
```
# HELP bbd_mineru mineru 调用情况
# TYPE bbd_mineru gauge
bbd_mineru{env="prod",region="ch-hangzhou"} 2443
```

### 3. Web UI 验证

打开指标管理页面 → 查看指标列表，确认 `bbd_mineru` 状态为启用（绿色电源图标）→ 在指标图表中选择 `bbd_mineru` 查看时序数据。
