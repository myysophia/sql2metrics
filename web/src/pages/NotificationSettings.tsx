import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/api/client'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Loader2, CheckCircle2, XCircle, Send, ArrowRight, Rocket } from 'lucide-react'
import type { NotifierConfig, WeChatNotifierConfig, DingTalkNotifierConfig, FeishuNotifierConfig } from '@/types/config'

export default function NotificationSettings() {
  const { toast } = useToast()
  const navigate = useNavigate()
  const [config, setConfig] = useState<NotifierConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState<string | null>(null)

  useEffect(() => {
    loadConfig()
  }, [])

  const loadConfig = async () => {
    try {
      const data = await api.getNotifierConfig()
      setConfig(data)
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

  const saveConfig = async () => {
    if (!config) return

    setSaving(true)
    try {
      await api.updateNotifierConfig(config)
      toast({
        title: '保存成功',
        description: '通知配置已更新，重启服务后生效',
      })
    } catch (error) {
      toast({
        title: '保存失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    } finally {
      setSaving(false)
    }
  }

  const testWebhook = async (channel: 'wechat' | 'dingtalk' | 'feishu', webhook: string, secret?: string) => {
    setTesting(channel)
    try {
      const result = await api.testNotifierWebhook(channel, webhook, secret)
      if (result.success) {
        toast({
          title: '测试成功',
          description: result.message || '测试消息已发送',
        })
      } else {
        toast({
          title: '测试失败',
          description: result.error || '发送失败',
          variant: 'destructive',
        })
      }
    } catch (error) {
      toast({
        title: '测试失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    } finally {
      setTesting(null)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!config) {
    return <div>加载配置失败</div>
  }

  return (
    <div className="space-y-6">
      {/* 头部 */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">通知设置</h2>
          <p className="text-muted-foreground">
            配置内置通知服务，支持企业微信、钉钉、飞书
          </p>
        </div>
        <div className="flex items-center gap-2">
          {/* 当前状态指示器 */}
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-md bg-muted">
            {config.enabled ? (
              <>
                <CheckCircle2 className="h-4 w-4 text-green-500" />
                <span className="text-sm">使用内置通知服务</span>
              </>
            ) : (
              <>
                <XCircle className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">使用外部 Alertmanager</span>
              </>
            )}
          </div>
          <Button onClick={saveConfig} disabled={saving}>
            {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            保存配置
          </Button>
        </div>
      </div>

      {/* 主开关 */}
      <Card>
        <CardHeader>
          <CardTitle>启用内置通知服务</CardTitle>
          <CardDescription>
            开启后将使用内置通知服务发送告警，关闭后将使用外部 Alertmanager
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <Label htmlFor="notifier-enabled">启用内置通知服务</Label>
            <Switch
              id="notifier-enabled"
              checked={config.enabled}
              onCheckedChange={(checked) =>
                setConfig({ ...config, enabled: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      {/* 顶部横幅：引导到高级路由 */}
      <Card className="bg-gradient-to-r from-blue-50 to-indigo-50 border-blue-200">
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-blue-100 rounded-lg">
                <Rocket className="h-6 w-6 text-blue-600" />
              </div>
              <div>
                <h3 className="font-semibold text-blue-900 text-lg">升级到智能路由系统</h3>
                <p className="text-sm text-blue-700 mt-1">
                  支持多渠道分发、基于标签和严重级别的智能路由、20+ 个 webhook 管理
                </p>
              </div>
            </div>
            <Button
              onClick={() => navigate('/routes')}
              className="bg-blue-600 hover:bg-blue-700"
            >
              前往告警路由
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 分组设置 */}
      {config.enabled && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>分组设置</CardTitle>
              <CardDescription>
                配置告警分组的等待时间和间隔
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="group-wait">首次等待时间</Label>
                  <Input
                    id="group-wait"
                    placeholder="30s"
                    value={config.group_wait || ''}
                    onChange={(e) =>
                      setConfig({ ...config, group_wait: e.target.value })
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    首次通知前的等待时间，如 30s, 5m
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="group-interval">分组间隔</Label>
                  <Input
                    id="group-interval"
                    placeholder="5m"
                    value={config.group_interval || ''}
                    onChange={(e) =>
                      setConfig({ ...config, group_interval: e.target.value })
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    同一组告警的通知间隔，如 5m, 10m
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="repeat-interval">重复间隔</Label>
                  <Input
                    id="repeat-interval"
                    placeholder="12h"
                    value={config.repeat_interval || ''}
                    onChange={(e) =>
                      setConfig({ ...config, repeat_interval: e.target.value })
                    }
                  />
                  <p className="text-xs text-muted-foreground">
                    重复发送同一告警的间隔，如 12h, 24h
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Separator />

          {/* 企业微信配置 */}
          <WeChatConfigSection
            config={config.wechat}
            onChange={(wechat) => setConfig({ ...config, wechat })}
            onTest={(webhook) => testWebhook('wechat', webhook)}
            testing={testing === 'wechat'}
          />

          <Separator />

          {/* 钉钉配置 */}
          <DingTalkConfigSection
            config={config.dingtalk}
            onChange={(dingtalk) => setConfig({ ...config, dingtalk })}
            onTest={(webhook, secret) => testWebhook('dingtalk', webhook, secret)}
            testing={testing === 'dingtalk'}
          />

          <Separator />

          {/* 飞书配置 */}
          <FeishuConfigSection
            config={config.feishu}
            onChange={(feishu) => setConfig({ ...config, feishu })}
            onTest={(webhook) => testWebhook('feishu', webhook)}
            testing={testing === 'feishu'}
          />

          {/* 底部提示：详细功能介绍 */}
          <Card className="border-dashed bg-muted/50">
            <CardContent className="pt-6">
              <div className="flex items-start gap-4">
                <div className="text-4xl">💡</div>
                <div className="flex-1">
                  <h3 className="font-semibold mb-2">需要更强大的路由功能？</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    告警路由系统支持：
                  </p>
                  <ul className="text-sm text-muted-foreground space-y-1 mb-4">
                    <li>• 配置 20+ 个不同的通知渠道</li>
                    <li>• 基于严重级别的智能分发</li>
                    <li>• 同一告警发送到多个群</li>
                    <li>• 按标签、告警名称灵活路由</li>
                  </ul>
                  <Button onClick={() => navigate('/routes')}>
                    前往告警路由管理
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

// 企业微信配置组件
function WeChatConfigSection({
  config,
  onChange,
  onTest,
  testing,
}: {
  config?: WeChatNotifierConfig
  onChange: (config: WeChatNotifierConfig) => void
  onTest: (webhook: string) => void
  testing: boolean
}) {
  const enabled = config?.enabled || false

  const updateConfig = (updates: Partial<WeChatNotifierConfig>) => {
    onChange({
      enabled: updates.enabled ?? enabled,
      webhook: updates.webhook ?? config?.webhook ?? '',
      mentioned_list: updates.mentioned_list ?? config?.mentioned_list ?? [],
      mentioned_mobile_list: updates.mentioned_mobile_list ?? config?.mentioned_mobile_list ?? [],
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>企业微信</CardTitle>
            <CardDescription>
              配置企业微信群机器人的 Webhook 地址
            </CardDescription>
          </div>
          <Switch
            checked={enabled}
            onCheckedChange={(checked) => updateConfig({ enabled: checked })}
          />
        </div>
      </CardHeader>
      {enabled && (
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="wechat-webhook">Webhook URL</Label>
            <Input
              id="wechat-webhook"
              placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY"
              value={config?.webhook || ''}
              onChange={(e) => updateConfig({ webhook: e.target.value })}
            />
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => onTest(config?.webhook || '')}
                disabled={!config?.webhook || testing}
              >
                {testing ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Send className="mr-2 h-4 w-4" />
                )}
                发送测试
              </Button>
              <span className="text-xs text-muted-foreground">
                发送测试消息到企业微信群
              </span>
            </div>
          </div>

          <div className="space-y-2">
            <Label>@的用户列表</Label>
            <Input
              placeholder="@all"
              value={config?.mentioned_list?.join(', ') || ''}
              onChange={(e) =>
                updateConfig({
                  mentioned_list: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
            />
            <p className="text-xs text-muted-foreground">
              多个用户用逗号分隔，如 @all 或 user1,user2
            </p>
          </div>

          <div className="space-y-2">
            <Label>@的手机号列表</Label>
            <Input
              placeholder="+86-13800000000"
              value={config?.mentioned_mobile_list?.join(', ') || ''}
              onChange={(e) =>
                updateConfig({
                  mentioned_mobile_list: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
            />
            <p className="text-xs text-muted-foreground">
              多个手机号用逗号分隔
            </p>
          </div>
        </CardContent>
      )}
    </Card>
  )
}

// 钉钉配置组件
function DingTalkConfigSection({
  config,
  onChange,
  onTest,
  testing,
}: {
  config?: DingTalkNotifierConfig
  onChange: (config: DingTalkNotifierConfig) => void
  onTest: (webhook: string, secret?: string) => void
  testing: boolean
}) {
  const enabled = config?.enabled || false

  const updateConfig = (updates: Partial<DingTalkNotifierConfig>) => {
    onChange({
      enabled: updates.enabled ?? enabled,
      webhook: updates.webhook ?? config?.webhook ?? '',
      secret: updates.secret ?? config?.secret,
      at_mobiles: updates.at_mobiles ?? config?.at_mobiles ?? [],
      at_user_ids: updates.at_user_ids ?? config?.at_user_ids ?? [],
      is_at_all: updates.is_at_all ?? config?.is_at_all ?? false,
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>钉钉</CardTitle>
            <CardDescription>
              配置钉钉群机器人的 Webhook 地址和签名密钥
            </CardDescription>
          </div>
          <Switch
            checked={enabled}
            onCheckedChange={(checked) => updateConfig({ enabled: checked })}
          />
        </div>
      </CardHeader>
      {enabled && (
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="dingtalk-webhook">Webhook URL</Label>
            <Input
              id="dingtalk-webhook"
              placeholder="https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
              value={config?.webhook || ''}
              onChange={(e) => updateConfig({ webhook: e.target.value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dingtalk-secret">签名密钥（可选）</Label>
            <Input
              id="dingtalk-secret"
              type="password"
              placeholder="SEC..."
              value={config?.secret || ''}
              onChange={(e) => updateConfig({ secret: e.target.value })}
            />
            <p className="text-xs text-muted-foreground">
              如果启用了签名验证，请填写加签密钥
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onTest(config?.webhook || '', config?.secret)}
              disabled={!config?.webhook || testing}
            >
              {testing ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Send className="mr-2 h-4 w-4" />
              )}
              发送测试
            </Button>
            <span className="text-xs text-muted-foreground">
              发送测试消息到钉钉群
            </span>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label>@的手机号列表</Label>
            <Input
              placeholder="13800000000"
              value={config?.at_mobiles?.join(', ') || ''}
              onChange={(e) =>
                updateConfig({
                  at_mobiles: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
            />
            <p className="text-xs text-muted-foreground">
              多个手机号用逗号分隔
            </p>
          </div>

          <div className="space-y-2">
            <Label>@的用户 ID 列表</Label>
            <Input
              placeholder="user123,user456"
              value={config?.at_user_ids?.join(', ') || ''}
              onChange={(e) =>
                updateConfig({
                  at_user_ids: e.target.value
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean),
                })
              }
            />
            <p className="text-xs text-muted-foreground">
              多个用户 ID 用逗号分隔
            </p>
          </div>

          <div className="flex items-center justify-between">
            <Label htmlFor="dingtalk-is-at-all">@所有人</Label>
            <Switch
              id="dingtalk-is-at-all"
              checked={config?.is_at_all || false}
              onCheckedChange={(checked) => updateConfig({ is_at_all: checked })}
            />
          </div>
        </CardContent>
      )}
    </Card>
  )
}

// 飞书配置组件
function FeishuConfigSection({
  config,
  onChange,
  onTest,
  testing,
}: {
  config?: FeishuNotifierConfig
  onChange: (config: FeishuNotifierConfig) => void
  onTest: (webhook: string) => void
  testing: boolean
}) {
  const enabled = config?.enabled || false

  const updateConfig = (updates: Partial<FeishuNotifierConfig>) => {
    onChange({
      enabled: updates.enabled ?? enabled,
      webhook: updates.webhook ?? config?.webhook ?? '',
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>飞书</CardTitle>
            <CardDescription>
              配置飞书群机器人的 Webhook 地址
            </CardDescription>
          </div>
          <Switch
            checked={enabled}
            onCheckedChange={(checked) => updateConfig({ enabled: checked })}
          />
        </div>
      </CardHeader>
      {enabled && (
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="feishu-webhook">Webhook URL</Label>
            <Input
              id="feishu-webhook"
              placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/YOUR_HOOK"
              value={config?.webhook || ''}
              onChange={(e) => updateConfig({ webhook: e.target.value })}
            />
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => onTest(config?.webhook || '')}
                disabled={!config?.webhook || testing}
              >
                {testing ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <Send className="mr-2 h-4 w-4" />
                )}
                发送测试
              </Button>
              <span className="text-xs text-muted-foreground">
                发送测试消息到飞书群
              </span>
            </div>
          </div>
        </CardContent>
      )}
    </Card>
  )
}
