import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import Editor from '@monaco-editor/react'
import { api } from '../api/client'
import type { Config, MetricSpec } from '../types/config'

interface MetricFormProps {
  metric: MetricSpec
  config: Config
  onSave: () => void
  onCancel: () => void
}

export default function MetricForm({ metric: initialMetric, config, onSave, onCancel }: MetricFormProps) {
  const [metric, setMetric] = useState(initialMetric)
  const [previewing, setPreviewing] = useState(false)
  const [previewResult, setPreviewResult] = useState<{ success: boolean; value?: number; error?: string } | null>(null)

  const queryClient = useQueryClient()

  const previewMutation = useMutation({
    mutationFn: () =>
      api.previewQuery({
        source: metric.source,
        query: metric.query,
        connection: metric.connection,
        result_field: metric.result_field,
      }),
    onMutate: () => {
      setPreviewing(true)
      setPreviewResult(null)
    },
    onSuccess: (result) => {
      setPreviewResult(result)
      setPreviewing(false)
    },
    onError: (error: Error) => {
      setPreviewResult({ success: false, error: error.message })
      setPreviewing(false)
    },
  })

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (initialMetric.name && initialMetric.name === metric.name) {
        return api.updateMetric(metric.name, metric)
      } else {
        return api.createMetric(metric)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      onSave()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    saveMutation.mutate()
  }

  const mysqlConnections = Object.keys(config.mysql_connections || {})
  const httpapiConnections = Object.keys(config.http_api_connections || {})

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            指标名称 <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={metric.name}
            onChange={(e) => setMetric({ ...metric, name: e.target.value })}
            required
            pattern="[a-zA-Z_:][a-zA-Z0-9_:]*"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            placeholder="energy_household_total"
          />
          <p className="mt-1 text-xs text-gray-500">只能包含字母、数字、下划线和冒号</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            指标类型 <span className="text-red-500">*</span>
          </label>
          <select
            value={metric.type}
            onChange={(e) => setMetric({ ...metric, type: e.target.value as MetricSpec['type'] })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          >
            <option value="gauge">Gauge</option>
            <option value="counter">Counter</option>
            <option value="histogram">Histogram</option>
            <option value="summary">Summary</option>
          </select>
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          帮助信息 <span className="text-red-500">*</span>
        </label>
        <input
          type="text"
          value={metric.help}
          onChange={(e) => setMetric({ ...metric, help: e.target.value })}
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          placeholder="户储设备总量"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            数据源 <span className="text-red-500">*</span>
          </label>
          <select
            value={metric.source}
            onChange={(e) => setMetric({ ...metric, source: e.target.value as 'mysql' | 'iotdb' | 'http_api' })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          >
            <option value="mysql">MySQL</option>
            <option value="iotdb">IoTDB</option>
            <option value="http_api">HTTP API</option>
          </select>
        </div>

        {metric.source === 'mysql' && mysqlConnections.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">连接</label>
            <select
              value={metric.connection || 'default'}
              onChange={(e) => setMetric({ ...metric, connection: e.target.value || undefined })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            >
              {mysqlConnections.map((conn) => (
                <option key={conn} value={conn}>
                  {conn}
                </option>
              ))}
            </select>
          </div>
        )}

        {metric.source === 'http_api' && httpapiConnections.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">连接</label>
            <select
              value={metric.connection || 'default'}
              onChange={(e) => setMetric({ ...metric, connection: e.target.value || undefined })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            >
              {httpapiConnections.map((conn) => (
                <option key={conn} value={conn}>
                  {conn}
                </option>
              ))}
            </select>
          </div>
        )}

        {(metric.source === 'iotdb' || metric.source === 'http_api') && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {metric.source === 'http_api' ? 'JSON 路径' : '结果字段'} <span className="text-red-500">{metric.source === 'http_api' ? '*' : ''}</span>
            </label>
            <input
              type="text"
              value={metric.result_field || ''}
              onChange={(e) => setMetric({ ...metric, result_field: e.target.value || undefined })}
              required={metric.source === 'http_api'}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
              placeholder={metric.source === 'http_api' ? 'main.mqttAuthUrl' : '留空则使用第一列'}
            />
            {metric.source === 'http_api' && (
              <p className="mt-1 text-xs text-gray-500">使用点号分隔的路径，如 main.mqttAuthUrl</p>
            )}
          </div>
        )}
      </div>

      {metric.source === 'http_api' ? (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            查询 URL <span className="text-red-500">*</span>
          </label>
          <input
            type="url"
            value={metric.query}
            onChange={(e) => setMetric({ ...metric, query: e.target.value })}
            required
            placeholder="https://control.pingjl.com/mqtt/control/v1/"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
          <p className="mt-1 text-xs text-gray-500">如果为空，将使用连接配置中的 URL</p>
        </div>
      ) : (
        <div>
          <div className="flex justify-between items-center mb-2">
            <label className="block text-sm font-medium text-gray-700">
              SQL 查询 <span className="text-red-500">*</span>
            </label>
            <button
              type="button"
              onClick={() => previewMutation.mutate()}
              disabled={previewing || !metric.query}
              className="px-3 py-1 text-sm bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50 focus-visible-ring"
            >
              {previewing ? '预览中…' : '预览查询'}
            </button>
          </div>
          <div className="border border-gray-300 rounded-md overflow-hidden">
            <Editor
              height="200px"
              defaultLanguage="sql"
              value={metric.query}
              onChange={(value) => setMetric({ ...metric, query: value || '' })}
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
              }}
            />
          </div>
        </div>
      )}
      
      {metric.source === 'http_api' && (
        <div>
          <button
            type="button"
            onClick={() => previewMutation.mutate()}
            disabled={previewing || !metric.result_field}
            className="px-3 py-1 text-sm bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50 focus-visible-ring"
          >
            {previewing ? '预览中…' : '预览查询'}
          </button>
        </div>
      )}
        {previewResult && (
          <div className={`mt-2 p-2 rounded ${previewResult.success ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'}`}>
            {previewResult.success ? (
              <div>查询结果: <span className="font-mono font-bold">{previewResult.value}</span></div>
            ) : (
              <div>错误: {previewResult.error}</div>
            )}
          </div>
        )}
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">标签</label>
        <div className="space-y-2">
          {Object.entries(metric.labels || {}).map(([key, value], index) => (
            <div key={index} className="flex space-x-2">
              <input
                type="text"
                value={key}
                onChange={(e) => {
                  const newLabels = { ...metric.labels }
                  delete newLabels[key]
                  newLabels[e.target.value] = value
                  setMetric({ ...metric, labels: newLabels })
                }}
                placeholder="标签键"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
              />
              <input
                type="text"
                value={value}
                onChange={(e) => {
                  setMetric({
                    ...metric,
                    labels: { ...metric.labels, [key]: e.target.value },
                  })
                }}
                placeholder="标签值"
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
              />
              <button
                type="button"
                onClick={() => {
                  const newLabels = { ...metric.labels }
                  delete newLabels[key]
                  setMetric({ ...metric, labels: newLabels })
                }}
                className="px-3 py-2 text-red-600 hover:text-red-700 focus-visible-ring"
              >
                删除
              </button>
            </div>
          ))}
          <button
            type="button"
            onClick={() => {
              setMetric({
                ...metric,
                labels: { ...(metric.labels || {}), '': '' },
              })
            }}
            className="text-sm text-primary-600 hover:text-primary-700 focus-visible-ring"
          >
            + 添加标签
          </button>
        </div>
      </div>

      {metric.type === 'histogram' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Buckets</label>
          <input
            type="text"
            value={metric.buckets?.join(',') || ''}
            onChange={(e) => {
              const buckets = e.target.value
                .split(',')
                .map((s) => parseFloat(s.trim()))
                .filter((n) => !isNaN(n))
              setMetric({ ...metric, buckets: buckets.length > 0 ? buckets : undefined })
            }}
            placeholder="0.005, 0.01, 0.025, 0.05, 0.1, ..."
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
      )}

      {metric.type === 'summary' && (
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Objectives</label>
          <input
            type="text"
            value={
              metric.objectives
                ? Object.entries(metric.objectives)
                    .map(([k, v]) => `${k}:${v}`)
                    .join(', ')
                : ''
            }
            onChange={(e) => {
              const objectives: Record<number, number> = {}
              e.target.value.split(',').forEach((pair) => {
                const [k, v] = pair.split(':').map((s) => parseFloat(s.trim()))
                if (!isNaN(k) && !isNaN(v)) {
                  objectives[k] = v
                }
              })
              setMetric({
                ...metric,
                objectives: Object.keys(objectives).length > 0 ? objectives : undefined,
              })
            }}
            placeholder="0.5:0.05, 0.9:0.01, 0.99:0.001"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
      )}

      <div className="flex justify-end space-x-2">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50 focus-visible-ring"
        >
          取消
        </button>
        <button
          type="submit"
          disabled={saveMutation.isPending}
          className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 disabled:opacity-50 focus-visible-ring"
        >
          {saveMutation.isPending ? '保存中…' : '保存'}
        </button>
      </div>
    </form>
  )
}


