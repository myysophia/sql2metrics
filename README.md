# sql2metrics

## 项目简介
sql2metrics 是一个以配置驱动的 Prometheus 指标采集器，能够定时执行数据库查询（当前支持 MySQL 与 IoTDB），并将结果转换为 Prometheus Metrics 暴露在 `/metrics` 端点。适用于快速将现有业务 SQL 转换为可监控的时间序列，用于观察趋势、告警与容量分析。

## 核心特性
- **配置驱动**：全部指标、SQL、连接信息通过 YAML 描述，新增监控无需改动代码。
- **多数据源支持**：同一进程内可连接多个 MySQL 数据库及 IoTDB，按指标选择数据源。
- **Prometheus 兼容**：内置 HTTP Server 暴露指标，同时提供采集状态指标便于自监控。
- **安全性**：敏感凭据通过 `.env` 或环境变量注入，配置文件中仅保留占位符。

## 快速开始
1. 安装依赖：要求 Go 1.21+，并保证采集器所在机器可访问目标数据库。
2. 配置凭据：复制 `.env`（或创建新文件）填入 `MYSQL_USER`、`MYSQL_PASS`、`IOTDB_USER`、`IOTDB_PASS` 等敏感信息。
3. 编辑配置：修改 `configs/config.yml` 或单独创建环境专用文件，按需增删 `metrics` 项目。
4. 启动采集器：
   ```bash
   go run ./cmd/collector -config configs/config.yml
   # 浏览 http://localhost:8080/metrics 查看指标
   ```
5. 部署运行：可打包为容器镜像、以 systemd/Kubernetes CronJob 等方式运行，定时抓取 Prometheus 指标。

## 配置结构说明
- `schedule.interval`：采集周期，支持 `1h`、`30m` 等 Go duration 格式。
- `mysql_connections`：声明多个 MySQL 连接（可共用实例不同库），指标通过 `connection` 字段选择。
- `iotdb`：配置 IoTDB 连接信息与会话参数；`result_field` 指定解析字段，若留空则自动选择首列。
- `metrics`：描述每个指标的名称、帮助信息、查询 SQL、标签与数据源。

## 指标约定与扩展
- 建议以业务域为前缀命名指标，例如 `sql2metrics_household_online`，标签使用小写英文。
- 同一指标可附加多标签（如 `region`、`category`），Prometheus 抓取后即可用于维度分析。
- 新增指标只需追加一段配置，无需重新编译或部署代码。

## 运行与排查
- 自监控指标：`collector_errors_total`（失败次数）、`collector_last_success_timestamp_seconds`（最近成功时间）。
- 日志：执行每个指标会输出查询 SQL、执行耗时与结果，可快速定位慢查询或异常。
- 若发生连接失败或权限错误，请检查数据库连通性、账号权限、SQL 是否在目标环境可执行。

## 下一步规划
- 扩展更多数据源（如 PostgreSQL、REST API）。
- 引入插件式聚合函数，支持对结果做平均、环比等计算。
- 集成 CI/CD 与自动化测试覆盖更多配置场景。
