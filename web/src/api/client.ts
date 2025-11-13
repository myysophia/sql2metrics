import type { Config, MetricSpec, MySQLConfig, IoTDBConfig, HTTPAPIConfig, ReloadResult } from '../types/config'

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
  
  validateConfig: () =>
    request<{ valid: boolean; error?: string }>('/config/validate'),
  
  getMetricsURL: () =>
    request<{ url: string }>('/config/metrics-url'),

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
  
  testHTTPAPI: (config: HTTPAPIConfig) =>
    request<{ success: boolean; error?: string; message?: string }>('/datasource/test/http_api', {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  
  previewQuery: (params: {
    source: 'mysql' | 'iotdb' | 'http_api'
    query: string
    connection?: string
    result_field?: string
    mysql_config?: MySQLConfig
    iotdb_config?: IoTDBConfig
    http_api_config?: HTTPAPIConfig
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
  
  deleteMetric: (name: string) =>
    request<{ message: string }>(`/metrics/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
}


