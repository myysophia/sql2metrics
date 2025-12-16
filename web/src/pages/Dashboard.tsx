import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Activity, Database, Server, Clock, ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'

export default function Dashboard() {
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  if (isLoading) {
    return <div className="text-center py-12">加载中…</div>
  }

  if (!config) {
    return <div className="text-center py-12 text-red-600">加载配置失败</div>
  }

  const listenHost = config.prometheus.listen_address === '0.0.0.0' ? 'localhost' : config.prometheus.listen_address
  const metricsURL = `http://${listenHost}:${config.prometheus.listen_port}/metrics`

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">概览</h2>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">指标总数</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums">{config.metrics.length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">MySQL 连接</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums">{Object.keys(config.mysql_connections || {}).length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Redis 连接</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums">{Object.keys(config.redis_connections || {}).length}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">采集周期</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{config.schedule.interval}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Metrics 端点</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-4">
            <code className="flex-1 bg-muted px-4 py-2 rounded font-mono text-sm">{metricsURL}</code>
            <Button asChild>
              <a
                href={metricsURL}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                打开
              </a>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
