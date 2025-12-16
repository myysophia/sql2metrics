import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { api } from '../api/client'
import type { Config } from '../types/config'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { Loader2 } from 'lucide-react'

export default function Settings() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [scheduleInterval, setScheduleInterval] = useState('')
  const [listenAddress, setListenAddress] = useState('')
  const [listenPort, setListenPort] = useState(8080)

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const { toast } = useToast()

  // Initialize state when config loads
  useEffect(() => {
    if (config && scheduleInterval === '') {
      setScheduleInterval(config.schedule.interval)
      setListenAddress(config.prometheus.listen_address)
      setListenPort(config.prometheus.listen_port)
    }
  }, [config, scheduleInterval])

  const updateMutation = useMutation({
    mutationFn: (newConfig: Config) => api.updateConfig(newConfig),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      toast({
        title: "配置已保存",
        description: "新的系统配置已成功应用",
      })
    },
    onError: (error: Error) => {
      toast({
        variant: "destructive",
        title: "保存失败",
        description: error.message,
      })
    },
  })

  const handleSave = () => {
    if (!config) return

    const newConfig: Config = {
      ...config,
      schedule: {
        interval: scheduleInterval,
      },
      prometheus: {
        listen_address: listenAddress,
        listen_port: listenPort,
      },
    }

    updateMutation.mutate(newConfig)
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">系统设置</h2>

      <Card>
        <CardHeader>
          <CardTitle>基础配置</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-2">
            <Label htmlFor="interval">采集周期</Label>
            <Input
              id="interval"
              type="text"
              value={scheduleInterval}
              onChange={(e) => setScheduleInterval(e.target.value)}
              placeholder="1h, 30m, 5m..."
            />
            <p className="text-sm text-muted-foreground">支持 Go duration 格式，如 1h、30m、5m</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="listenAddress">监听地址</Label>
            <Input
              id="listenAddress"
              type="text"
              value={listenAddress}
              onChange={(e) => setListenAddress(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="listenPort">监听端口</Label>
            <Input
              id="listenPort"
              type="number"
              value={listenPort}
              onChange={(e) => setListenPort(Number(e.target.value))}
            />
          </div>

          <div className="flex justify-end">
            <Button
              onClick={handleSave}
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  保存中…
                </>
              ) : (
                '保存'
              )}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
