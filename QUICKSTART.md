# SQL2Metrics 快速开始指南

## 一键启动

```bash
./start.sh start
```

## 手动启动

### 1. 配置数据源

编辑 `configs/config.yml`，配置你的数据源信息：

```yaml
mysql:
  host: "your-mysql-host"
  port: 3306
  user: "your-user"
  password: "your-password"
  database: "your-database"

redis:
  mode: "standalone"
  addr: "your-redis:6379"
  password: ""
```

### 2. 启动服务

```bash
# 开发模式
./start.sh dev

# 或生产模式（先构建）
./start.sh build
./sql2metrics -config=configs/config.yml
```

### 3. 访问 Web UI

打开浏览器访问：http://localhost:8080

### 4. 创建告警规则

1. 点击导航栏的 "告警"
2. 点击 "新建告警规则"
3. 填写告警配置：
   - 名称：设备数量高告警
   - 监控指标：选择一个指标
   - 条件类型：选择阈值/趋势/异常检测
   - 严重级别：critical/warning/info
4. 点击保存

## 配置 Alertmanager（可选）

### 安装 Alertmanager

```bash
# macOS
brew install alertmanager

# Linux
wget https://github.com/prometheus/alertmanager/releases/download/v0.26.0/alertmanager-0.26.0.linux-amd64.tar.gz
tar -xvf alertmanager-0.26.0.linux-amd64.tar.gz
sudo mv alertmanager-0.26.0.linux-amd64/alertmanager /usr/local/bin/
```

### 启动 Alertmanager

```bash
alertmanager --config.file=alertmanager.demo.yml
```

### 配置环境变量

```bash
export ALERTMANAGER_URL=http://localhost:9093
./sql2metrics -config=configs/config.yml
```

## 告警规则示例

### 阈值告警
当指标值 >= 1000 且持续 5 分钟时触发：

```json
{
  "name": "设备数量高告警",
  "metric_name": "energy_household_total",
  "condition": {
    "type": "threshold",
    "threshold": {
      "operator": ">=",
      "value": 1000,
      "duration": "5m"
    }
  },
  "severity": "warning"
}
```

### 趋势告警
当 1 小时内下降超过 50 时触发：

```json
{
  "name": "设备下降告警",
  "metric_name": "energy_household_online",
  "condition": {
    "type": "trend",
    "trend": {
      "type": "decrease",
      "window": "1h",
      "threshold": 50
    }
  },
  "severity": "critical"
}
```

### 异常检测告警
使用 Z-Score 算法检测统计异常（3σ）：

```json
{
  "name": "SOC 异常告警",
  "metric_name": "emsau_avgSoc",
  "condition": {
    "type": "anomaly",
    "anomaly": {
      "algorithm": "zscore",
      "window": "24h",
      "threshold": 3
    }
  },
  "severity": "warning"
}
```

## API 端点

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/alerts | 获取所有告警规则 |
| POST | /api/alerts | 创建告警规则 |
| PUT | /api/alerts/:id | 更新告警规则 |
| DELETE | /api/alerts/:id | 删除告警规则 |
| POST | /api/alerts/:id/enable | 启用告警 |
| POST | /api/alerts/:id/disable | 禁用告警 |
| GET | /api/alert-history | 获取告警历史 |
| POST | /api/alerts/evaluate | 手动触发评估 |

## 常见问题

### Q: 告警没有触发？
A: 检查以下几点：
1. 告警规则是否启用
2. 指标是否有数据（访问 /metrics 查看）
3. 条件设置是否正确
4. 查看日志输出

### Q: 如何查看告警历史？
A: 
- Web UI：告警列表 → 点击某个告警 → 查看详情
- API：GET /api/alert-history?rule_id=xxx

### Q: 如何测试告警规则？
A: 
- Web UI：告警详情页点击 "测试" 按钮
- API：POST /api/alerts/:id/test

### Q: 告警通知没收到？
A: 
1. 检查 Alertmanager 是否正常运行：http://localhost:9093
2. 检查 Alertmanager 配置是否正确
3. 查看 sql2metrics 日志确认是否发送成功

## 文件说明

| 文件 | 说明 |
|------|------|
| `configs/config.yml` | 主配置文件（数据源、指标定义） |
| `configs/alerts.json` | 告警规则存储 |
| `configs/config.demo.yml` | 配置示例 |
| `configs/alerts.json.demo` | 告警规则示例 |
| `alertmanager.demo.yml` | Alertmanager 配置示例 |
