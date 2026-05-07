// Route types for the alert routing system

export interface NotificationChannel {
  id: string
  name: string
  type: 'wechat' | 'dingtalk' | 'feishu' | 'sms' | 'call'
  enabled: boolean
  description?: string
  labels?: Record<string, string>
  created_at?: string
  updated_at?: string

  // Type-specific configurations
  wechat?: WeChatChannelConfig
  dingtalk?: DingTalkChannelConfig
  feishu?: FeishuChannelConfig
  sms?: any
  phone_call?: any
}

export interface WeChatChannelConfig {
  enabled: boolean
  webhook: string
  mentioned_list?: string[]
  mentioned_mobile_list?: string[]
}

export interface DingTalkChannelConfig {
  enabled: boolean
  webhook: string
  secret?: string
  at_mobiles?: string[]
  at_user_ids?: string[]
  is_at_all?: boolean
}

export interface FeishuChannelConfig {
  enabled: boolean
  webhook: string
}

export interface AlertRoute {
  id: string
  name: string
  enabled: boolean
  description?: string
  match: RouteMatch
  channel_ids: string[]
  continue: boolean
  priority: number
  created_at?: string
  updated_at?: string
}

export interface RouteMatch {
  labels?: Record<string, string>
  label_regex?: Record<string, string>
  severities?: string[]
  alert_names?: string
  alert_name_regex?: string
  metric_names?: string
  metric_name_regex?: string
}

// Channel type labels for display
export const CHANNEL_TYPE_LABELS: Record<string, string> = {
  wechat: '企业微信',
  dingtalk: '钉钉',
  feishu: '飞书',
  sms: '短信',
  call: '电话'
}

// Severity options
export const SEVERITY_OPTIONS = [
  { value: 'critical', label: '严重' },
  { value: 'warning', label: '警告' },
  { value: 'info', label: '信息' }
]
