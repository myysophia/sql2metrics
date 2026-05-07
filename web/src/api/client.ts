import type { Config, MetricSpec, MySQLConfig, IoTDBConfig, RedisConfig, RestAPIConfig, ReloadResult, NotifierConfig } from '../types/config'
import type { NotificationChannel, AlertRoute } from '../types/routes'

const API_BASE = '/api'

async function request<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  return response.json()
}

export const api = {
  // 配置管理
  getConfig: () => request<Config>('/config'),

  updateConfig: (config: Config) =>
    request<{ message: string; reload: ReloadResult }>('/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  validateConfig: () => request<{ valid: boolean; error?: string }>('/config/validate'),

  getMetricsURL: () => request<{ url: string }>('/config/metrics-url'),

  // 数据源测试
  testMySQL: (config: MySQLConfig) =>
    request<{ success: boolean; error?: string; message?: string }>('/datasource/test/mysql', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  testIoTDB: (config: IoTDBConfig) =>
    request<{ success: boolean; error?: string; message?: string }>('/datasource/test/iotdb', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  testRedis: (config: RedisConfig) =>
    request<{ success: boolean; error?: string; message?: string }>('/datasource/test/redis', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  testRestAPI: (config: RestAPIConfig) =>
    request<{ success: boolean; error?: string; message?: string }>('/datasource/test/restapi', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  previewRestAPI: (config: RestAPIConfig, query: string) =>
    request<{ success: boolean; data?: unknown; error?: string }>('/datasource/restapi/preview', {
      method: 'POST',
      body: JSON.stringify({ config, query }),
    }),

  previewQuery: (params: {
    source: 'mysql' | 'iotdb' | 'redis' | 'restapi'
    query: string
    connection?: string
    result_field?: string
    mysql_config?: MySQLConfig
    iotdb_config?: IoTDBConfig
    redis_config?: RedisConfig
    restapi_config?: RestAPIConfig
  }) =>
    request<{ success: boolean; value?: number; error?: string }>('/datasource/query/preview', {
      method: 'POST',
      body: JSON.stringify(params),
    }),

  // 指标管理
  listMetrics: () => request<MetricSpec[]>('/metrics'),

  getMetric: (name: string) => request<MetricSpec>(`/metrics/${encodeURIComponent(name)}`),

  createMetric: (metric: MetricSpec) =>
    request<MetricSpec>('/metrics', {
      method: 'POST',
      body: JSON.stringify(metric),
    }),

  updateMetric: (name: string, metric: MetricSpec) =>
    request<MetricSpec>(`/metrics/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(metric),
    }),

  updateMetricByIndex: (index: number, metric: MetricSpec) =>
    request<MetricSpec>(`/metrics/index/${index}`, {
      method: 'PUT',
      body: JSON.stringify(metric),
    }),

  deleteMetric: (name: string) =>
    request<{ message: string }>(`/metrics/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),

  deleteMetricByIndex: (index: number) =>
    request<{ message: string }>(`/metrics/index/${index}`, {
      method: 'DELETE',
    }),

  enableMetric: (index: number) =>
    request<MetricSpec>(`/metrics/index/${index}/enable`, {
      method: 'POST',
    }),

  disableMetric: (index: number) =>
    request<MetricSpec>(`/metrics/index/${index}/disable`, {
      method: 'POST',
    }),

  // ===================== 独立数据源 API =====================

  // MySQL
  updateMySQLConnection: (name: string, config: MySQLConfig) =>
    request<{ success: boolean; message: string }>(`/datasource/mysql/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  deleteMySQLConnection: (name: string) =>
    request<{ success: boolean; message: string }>(`/datasource/mysql/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),

  // Redis
  updateRedisConnection: (name: string, config: RedisConfig) =>
    request<{ success: boolean; message: string }>(`/datasource/redis/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  deleteRedisConnection: (name: string) =>
    request<{ success: boolean; message: string }>(`/datasource/redis/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),

  // RestAPI
  updateRestAPIConnection: (name: string, config: RestAPIConfig) =>
    request<{ success: boolean; message: string }>(`/datasource/restapi/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  deleteRestAPIConnection: (name: string) =>
    request<{ success: boolean; message: string }>(`/datasource/restapi/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),

  // IoTDB
  updateIoTDB: (config: IoTDBConfig) =>
    request<{ success: boolean; message: string }>('/datasource/iotdb', {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  // 新增指标
  addMetric: (metric: MetricSpec) =>
    request<{ success: boolean; message: string; index: number }>('/metrics/add', {
      method: 'POST',
      body: JSON.stringify(metric),
    }),

  // ===================== 告警 API =====================
  listAlerts: () => request<any[]>('/alerts'),

  getAlert: (id: string) => request<any>(`/alerts/${encodeURIComponent(id)}`),

  createAlert: (alert: any) =>
    request<any>('/alerts', {
      method: 'POST',
      body: JSON.stringify(alert),
    }),

  updateAlert: (id: string, alert: any) =>
    request<any>(`/alerts/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(alert),
    }),

  deleteAlert: (id: string) =>
    request<{ message: string }>(`/alerts/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),

  enableAlert: (id: string) =>
    request<{ message: string }>(`/alerts/${encodeURIComponent(id)}/enable`, {
      method: 'POST',
    }),

  disableAlert: (id: string) =>
    request<{ message: string }>(`/alerts/${encodeURIComponent(id)}/disable`, {
      method: 'POST',
    }),

  testAlert: (id: string) =>
    request<any>(`/alerts/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    }),

  getAlertHistory: (params?: { page?: number; page_size?: number; rule_id?: string }) => {
    const queryString = params ? `?${new URLSearchParams(params as any).toString()}` : ''
    return request<any>(`/alert-history${queryString}`)
  },

  evaluateAllAlerts: () =>
    request<any[]>('/alerts/evaluate', {
      method: 'POST',
    }),

  getAlertStats: () => request<any>('/alerts/stats'),

  // ===================== 通知配置 API =====================
  getNotifierConfig: () => request<NotifierConfig>('/notifier/config'),

  updateNotifierConfig: (config: NotifierConfig) =>
    request<{ message: string; reload?: ReloadResult }>('/notifier/config', {
      method: 'PUT',
      body: JSON.stringify(config),
    }),

  testNotifierWebhook: (channel: 'wechat' | 'dingtalk' | 'feishu', webhook: string, secret?: string) =>
    request<{ success: boolean; message?: string; error?: string }>('/notifier/test', {
      method: 'POST',
      body: JSON.stringify({ channel, webhook, secret }),
    }),

  // ===================== 时序数据查询 API =====================
  listAvailableMetrics: () => request<string[]>('/timeseries/metrics'),

  queryTimeseries: (params: {
    metrics: string[]
    start: string
    end?: string
    step?: string
  }) =>
    request<{
      data: Array<{
        metric: Record<string, string>
        values: [number, number][]
      }>
    }>('/timeseries/query', {
      method: 'POST',
      body: JSON.stringify(params),
    }),

  exportTimeseries: (params: {
    metrics: string[]
    start: string
    end?: string
    step?: string
  }) => {
    const queryParts: string[] = []
    params.metrics.forEach(m => queryParts.push(`metric=${encodeURIComponent(m)}`))
    queryParts.push(`start=${encodeURIComponent(params.start)}`)
    if (params.end) queryParts.push(`end=${encodeURIComponent(params.end)}`)
    if (params.step) queryParts.push(`step=${encodeURIComponent(params.step)}`)

    const queryString = queryParts.join('&')

    return fetch(`/api/timeseries/export?${queryString}`).then(resp => {
      if (!resp.ok) {
        throw new Error(`导出失败: ${resp.statusText}`)
      }
      return resp.blob()
    })
  },

  // ===================== 路由管理 API =====================

  // 通知渠道管理
  listNotificationChannels: () =>
    request<{ channels: NotificationChannel[]; total: number }>('/routes/channels')
      .then(r => r.channels),

  createNotificationChannel: (channel: NotificationChannel) =>
    request<NotificationChannel>('/routes/channels', {
      method: 'POST',
      body: JSON.stringify(channel),
    }),

  updateNotificationChannel: (id: string, channel: NotificationChannel) =>
    request<NotificationChannel>(`/routes/channels/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(channel),
    }),

  deleteNotificationChannel: (id: string) =>
    request<{ message: string; id: string }>(`/routes/channels/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),

  testNotificationChannel: (id: string) =>
    request<{ message: string; channel: string; channel_name: string }>(`/routes/channels/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    }),

  // 路由规则管理
  listAlertRoutes: () =>
    request<{ routes: AlertRoute[]; total: number }>('/routes/rules')
      .then(r => r.routes),

  createAlertRoute: (route: AlertRoute) =>
    request<AlertRoute>('/routes/rules', {
      method: 'POST',
      body: JSON.stringify(route),
    }),

  updateAlertRoute: (id: string, route: AlertRoute) =>
    request<AlertRoute>(`/routes/rules/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(route),
    }),

  deleteAlertRoute: (id: string) =>
    request<{ message: string; id: string }>(`/routes/rules/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    }),
}
