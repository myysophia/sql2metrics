import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

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
        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">指标总数</div>
          <div className="text-3xl font-bold tabular-nums">{config.metrics.length}</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">MySQL 连接</div>
          <div className="text-3xl font-bold tabular-nums">{Object.keys(config.mysql_connections || {}).length}</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">Redis 连接</div>
          <div className="text-3xl font-bold tabular-nums">{Object.keys(config.redis_connections || {}).length}</div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">采集周期</div>
          <div className="text-3xl font-bold">{config.schedule.interval}</div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Metrics 端点</h3>
        <div className="flex items-center space-x-4">
          <code className="flex-1 bg-gray-100 px-4 py-2 rounded">{metricsURL}</code>
          <a
            href={metricsURL}
            target="_blank"
            rel="noopener noreferrer"
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
          >
            打开
          </a>
        </div>
      </div>
    </div>
  )
}
