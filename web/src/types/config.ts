export interface ScheduleConfig {
  interval: string
}

export interface PrometheusConfig {
  listen_address: string
  listen_port: number
}

export interface MySQLConfig {
  host: string
  port: number
  user: string
  password: string
  database: string
  params?: Record<string, string>
}

export interface IoTDBConfig {
  host: string
  port: number
  user: string
  password: string
  fetch_size: number
  zone_id: string
  enable_tls: boolean
  enable_zstd: boolean
  session_pool?: number
}

export interface MetricSpec {
  name: string
  help: string
  type: 'gauge' | 'counter' | 'histogram' | 'summary'
  source: 'mysql' | 'iotdb'
  query: string
  labels?: Record<string, string>
  result_field?: string
  connection?: string
  buckets?: number[]
  objectives?: Record<number, number>
}

export interface Config {
  schedule: ScheduleConfig
  prometheus: PrometheusConfig
  mysql: MySQLConfig
  mysql_connections: Record<string, MySQLConfig>
  iotdb: IoTDBConfig
  metrics: MetricSpec[]
}

export interface ReloadResult {
  success: boolean
  error?: string
  message: string
  metrics?: string[]
  removed?: string[]
}


