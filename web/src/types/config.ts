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

export interface RedisConfig {
  mode: 'standalone' | 'sentinel' | 'cluster'
  addr: string
  username?: string
  password?: string
  db?: number
  enable_tls?: boolean
  skip_tls_verify?: boolean
}

export interface MetricSpec {
  name: string
  help: string
  type: 'gauge' | 'counter' | 'histogram' | 'summary'
  source: 'mysql' | 'iotdb' | 'redis' | 'restapi'
  query: string
  labels?: Record<string, string>
  result_field?: string
  connection?: string
  buckets?: number[]
  objectives?: Record<number, number>
}

export interface RestAPIConfig {
  base_url: string
  timeout?: string
  headers?: Record<string, string>
  tls?: {
    skip_verify?: boolean
  }
  retry?: {
    max_attempts?: number
    backoff?: string
  }
}

// 内置通知服务配置
export interface NotifierConfig {
  enabled: boolean
  group_wait?: string
  group_interval?: string
  repeat_interval?: string
  wechat?: WeChatNotifierConfig
  dingtalk?: DingTalkNotifierConfig
  feishu?: FeishuNotifierConfig
}

export interface WeChatNotifierConfig {
  enabled: boolean
  webhook: string
  mentioned_list?: string[]
  mentioned_mobile_list?: string[]
}

export interface DingTalkNotifierConfig {
  enabled: boolean
  webhook: string
  secret?: string
  at_mobiles?: string[]
  at_user_ids?: string[]
  is_at_all?: boolean
}

export interface FeishuNotifierConfig {
  enabled: boolean
  webhook: string
}

// Alertmanager 配置
export interface AlertmanagerConfig {
  url: string
}

export interface Config {
  schedule: ScheduleConfig
  prometheus: PrometheusConfig
  alertmanager: AlertmanagerConfig
  notifier?: NotifierConfig

  mysql: MySQLConfig
  mysql_connections: Record<string, MySQLConfig>

  redis: RedisConfig
  redis_connections: Record<string, RedisConfig>

  restapi_connections: Record<string, RestAPIConfig>

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
