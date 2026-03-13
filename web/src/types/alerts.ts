// Alert rule types
export interface AlertRule {
  id: string
  name: string
  description: string
  enabled: boolean
  metric_name: string
  evaluation_mode: 'collection' | 'scheduled'
  evaluation_interval?: string
  evaluation_interval_ms?: number
  condition: AlertCondition
  severity: 'critical' | 'warning' | 'info'
  labels: Record<string, string>
  annotations: Record<string, string>
  state: 'pending' | 'firing' | 'resolved'
  last_evaluation?: string
  last_triggered?: string
  trigger_count: number
  created_at: string
  updated_at: string
}

export interface AlertCondition {
  type: 'threshold' | 'trend' | 'anomaly'
  threshold?: ThresholdCondition
  trend?: TrendCondition
  anomaly?: AnomalyCondition
}

export interface ThresholdCondition {
  operator: '>' | '>=' | '<' | '<=' | '==' | '!='
  value: number
  duration?: string
}

export interface TrendCondition {
  type: 'increase' | 'decrease' | 'percentage_change'
  window: string
  window_ms?: number
  threshold: number
  comparison?: 'previous_window' | 'fixed_value'
}

export interface AnomalyCondition {
  algorithm: 'zscore' | 'iqr' | 'moving_average'
  window: string
  window_ms?: number
  threshold: number
  sensitivity?: 'low' | 'medium' | 'high'
}

export interface AlertHistory {
  id: string
  alert_rule_id: string
  alert_rule_name: string
  state: 'firing' | 'resolved'
  value: number
  message: string
  triggered_at: string
  resolved_at?: string
  labels: Record<string, string>
}

export interface AlertEvaluationResult {
  rule_id: string
  rule_name: string
  triggered: boolean
  value: number
  message: string
  evaluated_at: string
}

export interface AlertHistoryResponse {
  data: AlertHistory[]
  total: number
  page: number
  page_size: number
}

export interface AlertStats {
  total: number
  firing: number
  resolved: number
  total_rules: number
  enabled_rules: number
}
