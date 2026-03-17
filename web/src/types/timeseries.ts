// 时序数据查询请求
export interface TimeseriesQuery {
  metrics: string[]
  start: string  // "-1h" or ISO 8601
  end?: string
  step?: string
}

// 时序数据响应
export interface TimeseriesResponse {
  data: TimeseriesData[]
}

// 单条时序数据
export interface TimeseriesData {
  metric: Record<string, string>
  values: [number, number][]  // [timestamp, value]
}

// 图表数据点
export interface ChartDataPoint {
  timestamp: string
  [key: string]: number | string
}

// 预设时间范围
export type TimeRangePreset = '5m' | '15m' | '1h' | '6h' | '24h' | '7d'

// 时间范围选项
export interface TimeRangeOption {
  value: TimeRangePreset
  label: string
}

// 刷新间隔选项
export type RefreshInterval = 30 | 60 | 300  // seconds

// 导出格式
export type ExportFormat = 'csv' | 'png'
