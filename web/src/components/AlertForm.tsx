import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import type { Config } from '../types/config'
import type { AlertRule } from '../types/alerts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Loader2, Play } from 'lucide-react'
import AlertConditionBuilder from './AlertConditionBuilder'
import { toast } from '@/hooks/use-toast'

interface AlertFormProps {
  config: Config
  existingAlert?: AlertRule
  onSave: () => void
  onCancel: () => void
}

export default function AlertForm({ config, existingAlert, onSave, onCancel }: AlertFormProps) {
  const queryClient = useQueryClient()

  // 初始化标签和注解的原始文本
  const initialLabelsText = Object.entries(existingAlert?.labels || {})
    .map(([k, v]) => `${k}=${v}`)
    .join('\n')
  const initialAnnotationsText = Object.entries(existingAlert?.annotations || {})
    .map(([k, v]) => `${k}=${v}`)
    .join('\n')

  const [formData, setFormData] = useState<Partial<AlertRule>>({
    name: existingAlert?.name || '',
    description: existingAlert?.description || '',
    enabled: existingAlert?.enabled ?? true,
    metric_name: existingAlert?.metric_name || '',
    evaluation_mode: existingAlert?.evaluation_mode || 'collection',
    evaluation_interval: existingAlert?.evaluation_interval || '5m',
    condition: existingAlert?.condition || {
      type: 'threshold',
      threshold: {
        operator: '>=',
        value: 0,
      },
    },
    severity: existingAlert?.severity || 'warning',
    labels: existingAlert?.labels || {},
    annotations: existingAlert?.annotations || {},
  })

  // 保存标签和注解的原始文本
  const [labelsText, setLabelsText] = useState(initialLabelsText)
  const [annotationsText, setAnnotationsText] = useState(initialAnnotationsText)

  const testMutation = useMutation({
    mutationFn: () => api.testAlert(existingAlert!.id),
    onSuccess: (result) => {
      if (result.triggered) {
        toast({
          title: '告警测试成功',
          description: `告警会触发！当前值: ${result.value}, 消息: ${result.message}`,
        })
      } else {
        toast({
          title: '告警测试成功',
          description: `告警不会触发。当前值: ${result.value}, 消息: ${result.message}`,
        })
      }
    },
    onError: (error: Error) => {
      toast({
        title: '测试失败',
        description: error.message,
        variant: 'destructive',
      })
    },
  })

  const handleTest = () => {
    testMutation.mutate()
  }

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (existingAlert) {
        return api.updateAlert(existingAlert.id, formData)
      }
      return api.createAlert(formData as AlertRule)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      toast({
        title: existingAlert ? '告警规则已更新' : '告警规则已创建',
      })
      onSave()
    },
    onError: (error: Error) => {
      toast({
        title: '保存失败',
        description: error.message,
        variant: 'destructive',
      })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    saveMutation.mutate()
  }

  const availableMetrics = config.metrics || []

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* 基本信息 */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">基本信息</h3>
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="name">告警名称 *</Label>
            <Input
              id="name"
              required
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              placeholder="例如: 设备数量过高"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="severity">严重级别 *</Label>
            <Select
              value={formData.severity}
              onValueChange={(value) => setFormData({ ...formData, severity: value as 'critical' | 'warning' | 'info' })}
            >
              <SelectTrigger id="severity">
                <SelectValue placeholder="选择严重级别" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="critical">严重</SelectItem>
                <SelectItem value="warning">警告</SelectItem>
                <SelectItem value="info">信息</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="description">描述</Label>
          <Textarea
            id="description"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="告警规则的详细描述"
            rows={2}
          />
        </div>
      </div>

      {/* 指标和评估设置 */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">指标和评估</h3>
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="metric_name">指标名称 *</Label>
            <Select
              value={formData.metric_name}
              onValueChange={(value) => setFormData({ ...formData, metric_name: value })}
            >
              <SelectTrigger id="metric_name">
                <SelectValue placeholder="选择指标" />
              </SelectTrigger>
              <SelectContent>
                {availableMetrics.map((metric) => (
                  <SelectItem key={metric.name} value={metric.name}>
                    {metric.name} - {metric.help}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="evaluation_mode">评估模式 *</Label>
            <Select
              value={formData.evaluation_mode}
              onValueChange={(value) => setFormData({ ...formData, evaluation_mode: value as any })}
            >
              <SelectTrigger id="evaluation_mode">
                <SelectValue placeholder="选择评估模式" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="collection">采集时评估</SelectItem>
                <SelectItem value="scheduled">定时评估</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        {formData.evaluation_mode === 'scheduled' && (
          <div className="space-y-2">
            <Label htmlFor="evaluation_interval">评估间隔</Label>
            <Input
              id="evaluation_interval"
              value={formData.evaluation_interval}
              onChange={(e) => setFormData({ ...formData, evaluation_interval: e.target.value })}
              placeholder="例如: 5m, 1h"
            />
            <p className="text-xs text-muted-foreground">
              定时评估的间隔时间，例如: 30s, 5m, 1h
            </p>
          </div>
        )}
      </div>

      {/* 告警条件 */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">告警条件</h3>
        <AlertConditionBuilder
          condition={formData.condition || {
            type: 'threshold',
            threshold: {
              operator: '>=',
              value: 0,
            },
          }}
          onChange={(condition) => setFormData({ ...formData, condition })}
        />
      </div>

      {/* 标签和注解 */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold">标签和注解</h3>

        <div className="space-y-2">
          <Label>标签（可选）</Label>
          <div className="text-sm text-muted-foreground mb-2">
            添加自定义标签用于分组和路由，每行一个，格式：key=value（例如: team=operations）
          </div>
          <Textarea
            value={labelsText}
            onChange={(e) => {
              setLabelsText(e.target.value)
              // 同时更新 formData.labels
              const labels: Record<string, string> = {}
              e.target.value.split('\n').forEach((line) => {
                const trimmedLine = line.trim()
                if (trimmedLine && !trimmedLine.startsWith('#')) {
                  const eqIndex = trimmedLine.indexOf('=')
                  if (eqIndex > 0) {
                    const k = trimmedLine.substring(0, eqIndex).trim()
                    const v = trimmedLine.substring(eqIndex + 1).trim()
                    if (k) {
                      labels[k] = v
                    }
                  }
                }
              })
              setFormData({ ...formData, labels })
            }}
            placeholder="team=operations&#10;env=production&#10;region=ap-southeast-2"
            rows={3}
          />
        </div>

        <div className="space-y-2">
          <Label>注解（可选）</Label>
          <div className="text-sm text-muted-foreground mb-2">
            添加告警通知的额外信息，每行一个，格式：key=value（例如: summary=设备异常）
          </div>
          <Textarea
            value={annotationsText}
            onChange={(e) => {
              setAnnotationsText(e.target.value)
              // 同时更新 formData.annotations
              const annotations: Record<string, string> = {}
              e.target.value.split('\n').forEach((line) => {
                const trimmedLine = line.trim()
                if (trimmedLine && !trimmedLine.startsWith('#')) {
                  const eqIndex = trimmedLine.indexOf('=')
                  if (eqIndex > 0) {
                    const k = trimmedLine.substring(0, eqIndex).trim()
                    const v = trimmedLine.substring(eqIndex + 1).trim()
                    if (k) {
                      annotations[k] = v
                    }
                  }
                }
              })
              setFormData({ ...formData, annotations })
            }}
            placeholder="summary=设备数量过高&#10;runbook=https://docs.example.com/runbooks&#10;contact=ops@example.com"
            rows={3}
          />
        </div>
      </div>

      {/* 按钮 */}
      <div className="flex justify-end space-x-2">
        {existingAlert && (
          <Button
            type="button"
            variant="outline"
            onClick={handleTest}
            disabled={testMutation.isPending}
          >
            {testMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                测试中...
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                测试告警
              </>
            )}
          </Button>
        )}
        <Button type="button" variant="outline" onClick={onCancel} disabled={saveMutation.isPending}>
          取消
        </Button>
        <Button type="submit" disabled={saveMutation.isPending}>
          {saveMutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              保存中...
            </>
          ) : (
            '保存'
          )}
        </Button>
      </div>
    </form>
  )
}
