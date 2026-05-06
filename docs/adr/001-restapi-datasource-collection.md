# ADR-001: 补全 RestAPI 数据源的周期性采集链路

- **状态**: 已采纳 (Accepted)
- **日期**: 2026-05-06
- **关联**: `internal/collectors/service.go`, `internal/datasource/restapi.go`

## 背景

sql2metrics 支持通过 `restapi_connections` + `source: restapi` 配置 RESTful API 数据源，Web UI 提供了完整的连接管理、响应预览和 JSONPath 字段选择功能。`internal/datasource/restapi.go` 中 `RestAPIClient.QueryScalar()` 已实现 HTTP 请求执行和 JSON 数值提取。

但 `internal/collectors/service.go` 的采集调度（`queryMetric`）只处理了 `mysql`、`iotdb`、`redis` 三种数据源，**缺少 `restapi` 分支**。导致所有 restapi 类型的指标在周期采集时直接返回 `"数据源 restapi 未准备就绪"` 错误，Prometheus 导出的值始终为 0。

## 决策

在 `collectors/service.go` 中补全 restapi 数据源的完整生命周期管理，与其他数据源保持一致的架构模式：

1. **`Service` struct** 新增 `restapi map[string]*datasource.RestAPIClient` 字段
2. **`NewService`** 新增 RestAPI 客户端初始化循环（失败只警告，不阻止启动）
3. **`queryMetric`** 新增 `case "restapi"` 分支，调用 `client.QueryScalar(ctx, query, resultField)`
4. **`Close`** 新增 RestAPI 客户端关闭
5. **`ReloadConfig`** 新增 RestAPI 连接的清理、配置变更检测和重建
6. 新增辅助函数 `restapiConnectionsNeeded()` 和 `restapiConfigEqual()`

## 同时修复：Prometheus 热更新注册表冲突

在修复 RestAPI 采集后，通过 Web UI 修改指标 labels 触发热更新时，报错：

```
注册指标 ems_psc 失败: a previously registered descriptor with the same
fully-qualified name ... has different label names or a different help string
```

**根因**：

1. `ReloadConfig` 中 metric 变更检测条件只比较了 `Type` 和 `Labels`，**遗漏了 `Help`**。Prometheus descriptor 由 name + help + constLabels 三者共同确定，任一变化都会导致冲突。
2. 使用 `prometheus.Unregister()`（已废弃的包级函数）代替 `prometheus.DefaultRegisterer.Unregister()`，在某些场景下无法正确清理 descriptor。

**修复**：

1. 判断条件增加 `existingHolder.spec.Help != spec.Help`
2. 统一使用 `prometheus.DefaultRegisterer.Unregister()` / `MustRegister()` 进行注册表操作
3. `s.registry.Register` 失败时先 `Unregister` 已创建的 metric 再返回错误

## 影响

- RestAPI 数据源现在可被周期性采集，指标值正确暴露到 Prometheus `/metrics` 端点
- 热更新时修改指标的 labels 或 help 不再报注册冲突错误
- 无破坏性变更，原有 mysql/iotdb/redis 数据源行为不受影响
