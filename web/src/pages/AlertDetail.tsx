import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import AlertForm from '../components/AlertForm'
import type { Config } from '../types/config'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Loader2 } from 'lucide-react'

export default function AlertDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [isEditing, setIsEditing] = useState(false)

  const { data: config } = useQuery<Config>({
    queryKey: ['config'],
    queryFn: () => api.getConfig(),
  })

  const { data: alert, isLoading } = useQuery({
    queryKey: ['alerts', id],
    queryFn: () => api.getAlert(id!),
    enabled: !!id && id !== 'new',
  })

  if (id === 'new') {
    if (!config) return <div className="text-center py-12">加载中...</div>
    return (
      <div className="space-y-6">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/alerts')}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            返回
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-foreground">新建告警规则</h1>
            <p className="text-muted-foreground mt-1">创建新的指标告警规则</p>
          </div>
        </div>
        <AlertForm config={config} onSave={() => navigate('/alerts')} onCancel={() => navigate('/alerts')} />
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!alert || !config) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">告警规则不存在</p>
        <Button onClick={() => navigate('/alerts')}>返回列表</Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" onClick={() => navigate('/alerts')}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            返回
          </Button>
          <div>
            <h1 className="text-3xl font-bold text-foreground">{alert.name}</h1>
            <p className="text-muted-foreground mt-1">
              {alert.enabled ? '已启用' : '已禁用'} · {alert.state === 'pending' ? '待命' : alert.state === 'firing' ? '触发中' : '已恢复'}
            </p>
          </div>
        </div>
      </div>

      {!isEditing ? (
        <div className="border rounded-lg p-6 bg-background space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-muted-foreground">描述</p>
              <p className="mt-1">{alert.description || '-'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">监控指标</p>
              <p className="mt-1 font-mono">{alert.metric_name}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">评估模式</p>
              <p className="mt-1">{alert.evaluation_mode === 'collection' ? '采集时评估' : '定时评估'}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">严重级别</p>
              <p className="mt-1 capitalize">{alert.severity}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">条件类型</p>
              <p className="mt-1 capitalize">{alert.condition.type}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">触发次数</p>
              <p className="mt-1">{alert.trigger_count}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">最后触发</p>
              <p className="mt-1">
                {alert.last_triggered
                  ? new Date(alert.last_triggered).toLocaleString('zh-CN')
                  : '-'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">创建时间</p>
              <p className="mt-1">{new Date(alert.created_at).toLocaleString('zh-CN')}</p>
            </div>
          </div>

          <div>
            <p className="text-sm text-muted-foreground mb-2">标签</p>
            {Object.keys(alert.labels).length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {Object.entries(alert.labels).map(([k, v]) => (
                  <span
                    key={k}
                    className="px-2 py-1 bg-muted rounded text-sm font-mono"
                  >
                    {k}={String(v)}
                  </span>
                ))}
              </div>
            ) : (
              <p className="text-sm">-</p>
            )}
          </div>

          <div>
            <p className="text-sm text-muted-foreground mb-2">注解</p>
            {Object.keys(alert.annotations).length > 0 ? (
              <div className="space-y-1">
                {Object.entries(alert.annotations).map(([k, v]) => (
                  <div key={k} className="text-sm">
                    <span className="font-medium">{k}:</span> {String(v)}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm">-</p>
            )}
          </div>

          <div className="flex justify-end">
            <Button onClick={() => setIsEditing(true)}>编辑告警规则</Button>
          </div>
        </div>
      ) : (
        <AlertForm
          config={config}
          existingAlert={alert}
          onSave={() => {
            setIsEditing(false)
            navigate('/alerts')
          }}
          onCancel={() => setIsEditing(false)}
        />
      )}
    </div>
  )
}
