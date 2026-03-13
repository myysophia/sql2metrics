import type { AlertCondition, ThresholdCondition, TrendCondition, AnomalyCondition } from '../types/alerts'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface AlertConditionBuilderProps {
  condition: AlertCondition
  onChange: (condition: AlertCondition) => void
}

export default function AlertConditionBuilder({ condition, onChange }: AlertConditionBuilderProps) {
  const conditionType = condition.type || 'threshold'

  const handleTypeChange = (type: 'threshold' | 'trend' | 'anomaly') => {
    const newCondition: AlertCondition = { type }
    if (type === 'threshold') {
      newCondition.threshold = {
        operator: '>=',
        value: 0,
      }
    } else if (type === 'trend') {
      newCondition.trend = {
        type: 'increase',
        window: '1h',
        threshold: 0,
      }
    } else if (type === 'anomaly') {
      newCondition.anomaly = {
        algorithm: 'zscore',
        window: '24h',
        threshold: 3,
      }
    }
    onChange(newCondition)
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>告警条件类型</Label>
        <Select value={conditionType} onValueChange={handleTypeChange}>
          <SelectTrigger>
            <SelectValue placeholder="选择条件类型" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="threshold">阈值告警</SelectItem>
            <SelectItem value="trend">趋势告警</SelectItem>
            <SelectItem value="anomaly">异常检测</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {conditionType === 'threshold' && condition.threshold && (
        <ThresholdConditionForm
          condition={condition.threshold}
          onChange={(threshold) => onChange({ type: 'threshold', threshold })}
        />
      )}

      {conditionType === 'trend' && condition.trend && (
        <TrendConditionForm
          condition={condition.trend}
          onChange={(trend) => onChange({ type: 'trend', trend })}
        />
      )}

      {conditionType === 'anomaly' && condition.anomaly && (
        <AnomalyConditionForm
          condition={condition.anomaly}
          onChange={(anomaly) => onChange({ type: 'anomaly', anomaly })}
        />
      )}
    </div>
  )
}

interface ThresholdConditionFormProps {
  condition: ThresholdCondition
  onChange: (condition: ThresholdCondition) => void
}

function ThresholdConditionForm({ condition, onChange }: ThresholdConditionFormProps) {
  return (
    <div className="space-y-4 border rounded-lg p-4 bg-muted/50">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label>操作符</Label>
          <Select
            value={condition.operator}
            onValueChange={(value) => onChange({ ...condition, operator: value as any })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value=">">&gt; 大于</SelectItem>
              <SelectItem value=">=">&gt;= 大于等于</SelectItem>
              <SelectItem value="<">&lt; 小于</SelectItem>
              <SelectItem value="<=">&lt;= 小于等于</SelectItem>
              <SelectItem value="==">== 等于</SelectItem>
              <SelectItem value="!=">!= 不等于</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>阈值</Label>
          <Input
            type="number"
            step="any"
            value={condition.value}
            onChange={(e) => onChange({ ...condition, value: parseFloat(e.target.value) || 0 })}
            placeholder="输入阈值"
          />
        </div>

        <div className="space-y-2">
          <Label>持续时间（可选）</Label>
          <Input
            type="text"
            value={condition.duration || ''}
            onChange={(e) => onChange({ ...condition, duration: e.target.value || undefined })}
            placeholder="例如: 5m"
          />
          <p className="text-xs text-muted-foreground">
            告警条件需持续满足该时间才会触发（例如: 5m 表示 5 分钟）
          </p>
        </div>
      </div>
    </div>
  )
}

interface TrendConditionFormProps {
  condition: TrendCondition
  onChange: (condition: TrendCondition) => void
}

function TrendConditionForm({ condition, onChange }: TrendConditionFormProps) {
  return (
    <div className="space-y-4 border rounded-lg p-4 bg-muted/50">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label>趋势类型</Label>
          <Select
            value={condition.type}
            onValueChange={(value) => onChange({ ...condition, type: value as any })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="increase">增长</SelectItem>
              <SelectItem value="decrease">下降</SelectItem>
              <SelectItem value="percentage_change">百分比变化</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>时间窗口</Label>
          <Input
            type="text"
            value={condition.window}
            onChange={(e) => onChange({ ...condition, window: e.target.value })}
            placeholder="例如: 1h"
          />
          <p className="text-xs text-muted-foreground">分析该时间窗口内的数据变化</p>
        </div>

        <div className="space-y-2">
          <Label>阈值</Label>
          <Input
            type="number"
            step="any"
            value={condition.threshold}
            onChange={(e) => onChange({ ...condition, threshold: parseFloat(e.target.value) || 0 })}
            placeholder="输入阈值"
          />
          <p className="text-xs text-muted-foreground">
            {condition.type === 'percentage_change'
              ? '变化百分比阈值（例如: 10 表示 10%）'
              : '变化量阈值'}
          </p>
        </div>
      </div>
    </div>
  )
}

interface AnomalyConditionFormProps {
  condition: AnomalyCondition
  onChange: (condition: AnomalyCondition) => void
}

function AnomalyConditionForm({ condition, onChange }: AnomalyConditionFormProps) {
  return (
    <div className="space-y-4 border rounded-lg p-4 bg-muted/50">
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label>检测算法</Label>
          <Select
            value={condition.algorithm}
            onValueChange={(value) => onChange({ ...condition, algorithm: value as any })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="zscore">Z-Score (标准分数)</SelectItem>
              <SelectItem value="iqr">IQR (四分位距)</SelectItem>
              <SelectItem value="moving_average">移动平均</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>历史数据窗口</Label>
          <Input
            type="text"
            value={condition.window}
            onChange={(e) => onChange({ ...condition, window: e.target.value })}
            placeholder="例如: 24h"
          />
          <p className="text-xs text-muted-foreground">用于建立基线的历史数据时间范围</p>
        </div>

        <div className="space-y-2">
          <Label>检测阈值</Label>
          <Input
            type="number"
            step="any"
            value={condition.threshold}
            onChange={(e) => onChange({ ...condition, threshold: parseFloat(e.target.value) || 0 })}
            placeholder="输入阈值"
          />
          <p className="text-xs text-muted-foreground">
            {condition.algorithm === 'zscore'
              ? 'Z-Score 阈值（通常 3 表示 3σ）'
              : condition.algorithm === 'iqr'
              ? 'IQR 倍数（通常 1.5）'
              : '偏差百分比阈值'}
          </p>
        </div>
      </div>
    </div>
  )
}
