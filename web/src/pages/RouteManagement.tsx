import { useState, useEffect } from 'react'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/api/client'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Plus, Edit, Trash2, TestTube, Save } from 'lucide-react'
import type { NotificationChannel, AlertRoute } from '@/types/routes'
import { CHANNEL_TYPE_LABELS, SEVERITY_OPTIONS } from '@/types/routes'

export default function RouteManagement() {
  const { toast } = useToast()
  const [channels, setChannels] = useState<NotificationChannel[]>([])
  const [routes, setRoutes] = useState<AlertRoute[]>([])
  const [loading, setLoading] = useState(true)
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [editingRoute, setEditingRoute] = useState<AlertRoute | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [channelsData, routesData] = await Promise.all([
        api.listNotificationChannels(),
        api.listAlertRoutes()
      ])
      setChannels(channelsData)
      setRoutes(routesData)
    } catch (error) {
      toast({
        title: '加载失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleSaveChannel = async (channel: NotificationChannel) => {
    try {
      if (channel.id && channels.find(c => c.id === channel.id)) {
        await api.updateNotificationChannel(channel.id, channel)
        toast({ title: '成功', description: '渠道已更新' })
      } else {
        await api.createNotificationChannel(channel)
        toast({ title: '成功', description: '渠道已创建' })
      }
      setEditingChannel(null)
      loadData()
    } catch (error) {
      toast({
        title: '保存失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  const handleDeleteChannel = async (id: string) => {
    if (!confirm('确定要删除这个渠道吗？')) return

    try {
      await api.deleteNotificationChannel(id)
      toast({ title: '成功', description: '渠道已删除' })
      loadData()
    } catch (error) {
      toast({
        title: '删除失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  const handleTestChannel = async (id: string) => {
    try {
      const result = await api.testNotificationChannel(id)
      toast({ title: '成功', description: result.message })
    } catch (error) {
      toast({
        title: '测试失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  const handleSaveRoute = async (route: AlertRoute) => {
    try {
      if (route.id && routes.find(r => r.id === route.id)) {
        await api.updateAlertRoute(route.id, route)
        toast({ title: '成功', description: '路由规则已更新' })
      } else {
        await api.createAlertRoute(route)
        toast({ title: '成功', description: '路由规则已创建' })
      }
      setEditingRoute(null)
      loadData()
    } catch (error) {
      toast({
        title: '保存失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  const handleDeleteRoute = async (id: string) => {
    if (!confirm('确定要删除这条路由规则吗？')) return

    try {
      await api.deleteAlertRoute(id)
      toast({ title: '成功', description: '路由规则已删除' })
      loadData()
    } catch (error) {
      toast({
        title: '删除失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  if (loading) {
    return <div className="p-8">加载中...</div>
  }

  return (
    <div className="space-y-6 p-8">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold">告警路由管理</h2>
          <p className="text-muted-foreground">
            配置通知渠道和路由规则，实现智能告警分发
          </p>
        </div>
      </div>

      <Tabs defaultValue="channels" className="space-y-4">
        <TabsList>
          <TabsTrigger value="channels">
            通知渠道 ({channels.length})
          </TabsTrigger>
          <TabsTrigger value="routes">
            路由规则 ({routes.length})
          </TabsTrigger>
        </TabsList>

        <TabsContent value="channels" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>通知渠道</CardTitle>
                  <CardDescription>
                    管理企业微信、钉钉、飞书等通知渠道
                  </CardDescription>
                </div>
                <Button onClick={() => setEditingChannel({
                  id: '',
                  name: '',
                  type: 'wechat',
                  enabled: true,
                  wechat: { enabled: true, webhook: '' }
                } as NotificationChannel)}>
                  <Plus className="mr-2 h-4 w-4" />
                  添加渠道
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {channels.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  暂无通知渠道，点击上方按钮添加
                </div>
              ) : (
                <div className="space-y-3">
                  {channels.map((channel) => (
                    <div key={channel.id} className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50">
                      <div className="flex items-center gap-4">
                        <Badge variant={channel.enabled ? 'default' : 'secondary'}>
                          {CHANNEL_TYPE_LABELS[channel.type] || channel.type}
                        </Badge>
                        <div>
                          <div className="font-medium">{channel.name}</div>
                          <div className="text-sm text-muted-foreground">{channel.id}</div>
                          {channel.description && (
                            <div className="text-sm text-muted-foreground mt-1">{channel.description}</div>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={channel.enabled} disabled />
                        <Button variant="ghost" size="sm" onClick={() => setEditingChannel(channel)}>
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleTestChannel(channel.id)}>
                          <TestTube className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleDeleteChannel(channel.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {editingChannel && (
            <ChannelEditDialog
              channel={editingChannel}
              open={!!editingChannel}
              onClose={() => setEditingChannel(null)}
              onSave={handleSaveChannel}
            />
          )}
        </TabsContent>

        <TabsContent value="routes" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>路由规则</CardTitle>
                  <CardDescription>
                    配置告警匹配条件和目标渠道
                  </CardDescription>
                </div>
                <Button onClick={() => setEditingRoute({
                  id: '',
                  name: '',
                  enabled: true,
                  match: {},
                  channel_ids: [],
                  continue: false,
                  priority: 50
                } as AlertRoute)}>
                  <Plus className="mr-2 h-4 w-4" />
                  添加规则
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {routes.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  暂无路由规则，点击上方按钮添加
                </div>
              ) : (
                <div className="space-y-3">
                  {routes.map((route) => (
                    <div key={route.id} className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50">
                      <div className="flex items-center gap-4">
                        <Badge variant={route.enabled ? 'default' : 'secondary'}>
                          优先级: {route.priority}
                        </Badge>
                        <div className="flex-1">
                          <div className="font-medium">{route.name}</div>
                          <div className="text-sm text-muted-foreground mt-1">
                            {formatMatchConditions(route.match)} → {route.channel_ids.length} 个渠道
                          </div>
                          {route.description && (
                            <div className="text-sm text-muted-foreground mt-1">{route.description}</div>
                          )}
                        </div>
                        {route.continue && (
                          <Badge variant="outline">继续匹配</Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch checked={route.enabled} disabled />
                        <Button variant="ghost" size="sm" onClick={() => setEditingRoute(route)}>
                          <Edit className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => handleDeleteRoute(route.id)}>
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {editingRoute && (
            <RouteEditDialog
              route={editingRoute}
              channels={channels}
              open={!!editingRoute}
              onClose={() => setEditingRoute(null)}
              onSave={handleSaveRoute}
            />
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

// Helper function to format match conditions
function formatMatchConditions(match: AlertRoute['match']): string {
  const conditions: string[] = []

  if (match.severities && match.severities.length > 0) {
    conditions.push(`严重级别: ${match.severities.join(', ')}`)
  }
  if (match.alert_names) {
    conditions.push(`告警名称: ${match.alert_names}`)
  }
  if (match.metric_names) {
    conditions.push(`指标名称: ${match.metric_names}`)
  }
  if (match.labels && Object.keys(match.labels).length > 0) {
    const labelStr = Object.entries(match.labels)
      .map(([k, v]) => `${k}=${v}`)
      .join(', ')
    conditions.push(`标签: ${labelStr}`)
  }

  return conditions.length > 0 ? conditions.join(' | ') : '匹配所有告警'
}

// Channel Edit Dialog Component
function ChannelEditDialog({
  channel,
  open,
  onClose,
  onSave
}: {
  channel: NotificationChannel
  open: boolean
  onClose: () => void
  onSave: (channel: NotificationChannel) => void
}) {
  const [formData, setFormData] = useState<NotificationChannel>({ ...channel })

  if (!open) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave(formData)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader>
          <CardTitle>{channel.id ? '编辑渠道' : '添加渠道'}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">名称 *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="type">类型 *</Label>
                <select
                  id="type"
                  className="w-full px-3 py-2 border rounded-md"
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as any })}
                  required
                >
                  <option value="wechat">企业微信</option>
                  <option value="dingtalk">钉钉</option>
                  <option value="feishu">飞书</option>
                </select>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="webhook">Webhook URL *</Label>
              <Input
                id="webhook"
                type="url"
                placeholder={formData.type === 'wechat' ? 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx' :
                        formData.type === 'dingtalk' ? 'https://oapi.dingtalk.com/robot/send?access_token=xxx' :
                        'https://open.feishu.cn/open-apis/bot/v2/hook/xxx'}
                value={formData.type === 'wechat' ? formData.wechat?.webhook || '' :
                        formData.type === 'dingtalk' ? formData.dingtalk?.webhook || '' :
                        formData.feishu?.webhook || ''}
                onChange={(e) => {
                  const webhook = e.target.value
                  if (formData.type === 'wechat') {
                    setFormData({ ...formData, wechat: { ...formData.wechat, enabled: true, webhook } })
                  } else if (formData.type === 'dingtalk') {
                    setFormData({ ...formData, dingtalk: { ...formData.dingtalk, enabled: true, webhook } })
                  } else {
                    setFormData({ ...formData, feishu: { ...formData.feishu, enabled: true, webhook } })
                  }
                }}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={formData.description || ''}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="渠道的用途说明"
              />
            </div>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                取消
              </Button>
              <Button type="submit">
                <Save className="mr-2 h-4 w-4" />
                保存
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

// Route Edit Dialog Component
function RouteEditDialog({
  route,
  channels,
  open,
  onClose,
  onSave
}: {
  route: AlertRoute
  channels: NotificationChannel[]
  open: boolean
  onClose: () => void
  onSave: (route: AlertRoute) => void
}) {
  const [formData, setFormData] = useState<AlertRoute>({ ...route })

  if (!open) return null

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave(formData)
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <Card className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <CardHeader>
          <CardTitle>{route.id ? '编辑路由规则' : '添加路由规则'}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="name">名称 *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="priority">优先级 *</Label>
                <Input
                  id="priority"
                  type="number"
                  min="0"
                  max="100"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) })}
                  required
                />
                <p className="text-xs text-muted-foreground">数值越大优先级越高</p>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">描述</Label>
              <Input
                id="description"
                value={formData.description || ''}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              />
            </div>

            <div className="space-y-2">
              <Label>匹配条件</Label>
              <div className="border rounded-lg p-4 space-y-3">
                <div className="space-y-2">
                  <Label>严重级别</Label>
                  <div className="flex flex-wrap gap-2">
                    {SEVERITY_OPTIONS.map((sev) => (
                      <label key={sev.value} className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={formData.match.severities?.includes(sev.value) || false}
                          onChange={(e) => {
                            const severities = formData.match.severities || []
                            if (e.target.checked) {
                              setFormData({
                                ...formData,
                                match: { ...formData.match, severities: [...severities, sev.value] }
                              })
                            } else {
                              setFormData({
                                ...formData,
                                match: { ...formData.match, severities: severities.filter(s => s !== sev.value) }
                              })
                            }
                          }}
                        />
                        <span>{sev.label}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="alert_names">告警名称（逗号分隔）</Label>
                  <Input
                    id="alert_names"
                    placeholder="设备离线,服务异常"
                    value={formData.match.alert_names || ''}
                    onChange={(e) => setFormData({
                      ...formData,
                      match: { ...formData.match, alert_names: e.target.value }
                    })}
                  />
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <Label>目标渠道 *</Label>
              <div className="border rounded-lg p-4 space-y-2">
                {channels.length === 0 ? (
                  <p className="text-sm text-muted-foreground">暂无可用的通知渠道</p>
                ) : (
                  channels.map((channel) => (
                    <label key={channel.id} className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={formData.channel_ids.includes(channel.id)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setFormData({
                              ...formData,
                              channel_ids: [...formData.channel_ids, channel.id]
                            })
                          } else {
                            setFormData({
                              ...formData,
                              channel_ids: formData.channel_ids.filter(id => id !== channel.id)
                            })
                          }
                        }}
                      />
                      <span>{channel.name}</span>
                      <Badge variant="outline" className="text-xs">
                        {CHANNEL_TYPE_LABELS[channel.type]}
                      </Badge>
                    </label>
                  ))
                )}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="continue"
                checked={formData.continue}
                onChange={(e) => setFormData({ ...formData, continue: e.target.checked })}
              />
              <Label htmlFor="continue">继续匹配后续路由</Label>
            </div>

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={onClose}>
                取消
              </Button>
              <Button type="submit">
                <Save className="mr-2 h-4 w-4" />
                保存
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
