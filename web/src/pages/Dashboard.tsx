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

  // 安全访问嵌套属性
  const metrics = config.metrics || []
  const mysqlConnections = config.mysql_connections || {}
  const scheduleInterval = config.schedule?.interval || '未配置'
  const prometheusConfig = config.prometheus || { listen_address: '0.0.0.0', listen_port: 8080 }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">概览</h2>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">指标总数</div>
          <div className="text-3xl font-bold tabular-nums">{metrics.length}</div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">MySQL 连接</div>
          <div className="text-3xl font-bold tabular-nums">
            {Object.keys(mysqlConnections).length}
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <div className="text-sm text-gray-500 mb-1">采集周期</div>
          <div className="text-3xl font-bold">{scheduleInterval}</div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold mb-4">Metrics 端点</h3>
        <div className="flex items-center space-x-4">
          <code className="flex-1 bg-gray-100 px-4 py-2 rounded">
            http://{prometheusConfig.listen_address === '0.0.0.0' ? 'localhost' : prometheusConfig.listen_address}:
            {prometheusConfig.listen_port}/metrics
          </code>
          <a
            href={`http://${prometheusConfig.listen_address === '0.0.0.0' ? 'localhost' : prometheusConfig.listen_address}:${prometheusConfig.listen_port}/metrics`}
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


