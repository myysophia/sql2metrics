import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { Config } from '../types/config'

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

  if (scheduleInterval === '') {
    setScheduleInterval(config.schedule.interval)
    setListenAddress(config.prometheus.listen_address)
    setListenPort(config.prometheus.listen_port)
  }

  const updateMutation = useMutation({
    mutationFn: (newConfig: Config) => api.updateConfig(newConfig),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      alert('配置已保存')
    },
    onError: (error: Error) => {
      alert(`保存失败: ${error.message}`)
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

      <div className="bg-white rounded-lg shadow p-6 space-y-6">
        <div>
          <label htmlFor="interval" className="block text-sm font-medium text-gray-700 mb-2">
            采集周期
          </label>
          <input
            id="interval"
            type="text"
            value={scheduleInterval}
            onChange={(e) => setScheduleInterval(e.target.value)}
            placeholder="1h, 30m, 5m..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
          <p className="mt-1 text-sm text-gray-500">支持 Go duration 格式，如 1h、30m、5m</p>
        </div>

        <div>
          <label htmlFor="listenAddress" className="block text-sm font-medium text-gray-700 mb-2">
            监听地址
          </label>
          <input
            id="listenAddress"
            type="text"
            value={listenAddress}
            onChange={(e) => setListenAddress(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>

        <div>
          <label htmlFor="listenPort" className="block text-sm font-medium text-gray-700 mb-2">
            监听端口
          </label>
          <input
            id="listenPort"
            type="number"
            value={listenPort}
            onChange={(e) => setListenPort(Number(e.target.value))}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>

        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={updateMutation.isPending}
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed focus-visible-ring"
          >
            {updateMutation.isPending ? '保存中…' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )
}


