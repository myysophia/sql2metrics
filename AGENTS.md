# Repository Guidelines

## 项目结构与模块组织
- `cmd/collector/` 存放主二进制入口，按区域拆分调度流程。
- `internal/metrics/` 聚合 Prometheus 指标构建逻辑，保持纯函数便于单测。
- `configs/` 保存示例配置，例如 `configs/config.yml` 描述数据库与调度周期；敏感凭据使用环境变量映射。
- `scripts/` 汇总运维脚本，约定新脚本以 `region-task.sh` 命名并保留执行说明。

## 构建、测试与本地运行命令
- `make bootstrap` 安装 Go 依赖与预提交钩子；首次拉仓库后执行。
- `make run CONFIG=configs/config.dev.yml` 启动本地采集服务，默认监听 `:8080` 暴露 `/metrics`。
- `make test` 运行全部单元测试与静态检查；CI 将拦截失败用例。
- `make lint` 调用 `golangci-lint` 校验代码风格，提交前确保通过。

## 代码风格与命名约定
- Go 源码统一使用 `gofmt`；保持 4 空格缩进，禁止混用 Tab。
- 包命名采用短小复数，例如 `connectors`、`collectors`；文件名以功能命名如 `iotdb_client.go`。
- 配置键使用蛇形命名（如 `iotdb.query`），Prometheus 标签使用小写短横线（如 `region="china"`）。
- 复杂逻辑前添加简短中文注释，解释输入输出与关键边界条件。

## 测试规范
- 首选 `testing` 框架，必要时结合 `testify` 辅助断言。
- 单测文件以 `_test.go` 结尾，测试函数命名 `Test模块名_行为`。
- PR 必须覆盖新增逻辑的核心分支，目标语句覆盖率 ≥ 80%，可在 `make cover` 查看报告。

## 提交与合并请求指南
- Commit 信息遵循 `type(scope): subject`，type 推荐 `feat`、`fix`、`chore`、`docs`。
- PR 描述需列出变更摘要、测试结果、关联工单或 Issue 链接；涉及接口改动请附样例输出。
- 截图或日志仅保留必要关键信息，默认遮蔽敏感配置；评审人要求前不得合并。

## 安全与配置提示
- 所有数据库账号仅授予只读权限，密码通过环境变量 `MYSQL_PASS`、`IOTDB_PASS` 注入。
- 提交前检查 `configs/` 内无真实生产凭据；如需分享参考配置，使用 `.example` 后缀。
- 部署时结合 Prometheus 抓取或 Pushgateway 模式，保持 TLS/防火墙策略与公司安全基线一致。
