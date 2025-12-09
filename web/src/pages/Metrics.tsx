import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { Config, MetricSpec } from '../types/config'
import MetricForm from '../components/MetricForm'
import SaveAndApply from '../components/SaveAndApply'

export default function Metrics() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMetric, setEditingMetric] = useState<MetricSpec | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteMetric(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
    },
  })

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const handleDelete = (name: string) => {
    if (!confirm(`确定要删除指标 "${name}" 吗？`)) return
    deleteMutation.mutate(name)
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">指标管理</h2>
        <button
          onClick={() => {
            setIsCreating(true)
            setEditingMetric({
              name: '',
              help: '',
              type: 'gauge',
              source: 'mysql',
              query: '',
            })
          }}
          className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
        >
          添加指标
        </button>
      </div>

      {(editingMetric || isCreating) && (
        <div className="bg-white rounded-lg shadow p-6 mb-6">
          <h3 className="text-lg font-semibold mb-4">
            {isCreating ? '创建新指标' : '编辑指标'}
          </h3>
          <MetricForm
            metric={editingMetric!}
            config={config}
            onSave={() => {
              setEditingMetric(null)
              setIsCreating(false)
              queryClient.invalidateQueries({ queryKey: ['config'] })
            }}
            onCancel={() => {
              setEditingMetric(null)
              setIsCreating(false)
            }}
          />
        </div>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                名称
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                类型
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                数据源
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                帮助信息
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                操作
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {config.metrics.map((metric) => (
              <tr key={metric.name}>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  {metric.name}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {metric.type}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {metric.source}
                  {metric.connection && ` (${metric.connection})`}
                </td>
                <td className="px-6 py-4 text-sm text-gray-500">{metric.help}</td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <button
                    onClick={() => {
                      setEditingMetric(metric)
                      setIsCreating(false)
                    }}
                    className="text-primary-600 hover:text-primary-900 mr-4 focus-visible-ring"
                  >
                    编辑
                  </button>
                  <button
                    onClick={() => handleDelete(metric.name)}
                    className="text-red-600 hover:text-red-900 focus-visible-ring"
                  >
                    删除
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-6">
        <SaveAndApply />
      </div>
    </div>
  )
}

<<<<<<< HEAD

=======
>>>>>>> 59c5b8e (feat: redis)
