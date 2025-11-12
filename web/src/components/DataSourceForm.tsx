import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import type { MySQLConfig, IoTDBConfig } from '../types/config'

interface DataSourceFormProps {
  type: 'mysql' | 'iotdb'
  initialConfig: MySQLConfig | IoTDBConfig
  onSave: (config: MySQLConfig | IoTDBConfig) => void
  onCancel: () => void
}

export default function DataSourceForm({ type, initialConfig, onSave, onCancel }: DataSourceFormProps) {
  const [config, setConfig] = useState(initialConfig)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message?: string; error?: string } | null>(null)

  const testMutation = useMutation({
    mutationFn: async () => {
      if (type === 'mysql') {
        return api.testMySQL(config as MySQLConfig)
      } else {
        return api.testIoTDB(config as IoTDBConfig)
      }
    },
    onMutate: () => {
      setTesting(true)
      setTestResult(null)
    },
    onSuccess: (result) => {
      // 后端返回 success: false 时，也作为错误处理
      if (!result.success) {
        setTestResult({
          success: false,
          error: result.error || '连接失败',
          message: result.error || result.message || '连接失败',
        })
      } else {
        setTestResult(result)
      }
      setTesting(false)
    },
    onError: (error: Error) => {
      setTestResult({ success: false, message: error.message, error: error.message })
      setTesting(false)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSave(config)
  }

  if (type === 'mysql') {
    const mysqlConfig = config as MySQLConfig
    return (
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">主机</label>
          <input
            type="text"
            value={mysqlConfig.host}
            onChange={(e) => setConfig({ ...mysqlConfig, host: e.target.value })}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">端口</label>
          <input
            type="number"
            value={mysqlConfig.port}
            onChange={(e) => setConfig({ ...mysqlConfig, port: Number(e.target.value) })}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">用户</label>
          <input
            type="text"
            value={mysqlConfig.user}
            onChange={(e) => setConfig({ ...mysqlConfig, user: e.target.value })}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">密码</label>
          <input
            type="password"
            value={mysqlConfig.password}
            onChange={(e) => setConfig({ ...mysqlConfig, password: e.target.value })}
            placeholder="使用环境变量 ${MYSQL_PASS}"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">数据库</label>
          <input
            type="text"
            value={mysqlConfig.database}
            onChange={(e) => setConfig({ ...mysqlConfig, database: e.target.value })}
            required
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div className="flex items-center space-x-4">
          <button
            type="button"
            onClick={() => testMutation.mutate()}
            disabled={testing}
            className="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50 focus-visible-ring"
          >
            {testing ? '测试中…' : '测试连接'}
          </button>
          {testResult && (
            <span className={testResult.success ? 'text-green-600' : 'text-red-600'}>
              {testResult.success 
                ? `✓ ${testResult.message || '连接成功'}` 
                : `✗ ${testResult.error || testResult.message || '连接失败'}`}
            </span>
          )}
        </div>
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
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
          >
            保存
          </button>
        </div>
      </form>
    )
  }

  const iotdbConfig = config as IoTDBConfig
  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">主机</label>
        <input
          type="text"
          value={iotdbConfig.host}
          onChange={(e) => setConfig({ ...iotdbConfig, host: e.target.value })}
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">端口</label>
        <input
          type="number"
          value={iotdbConfig.port}
          onChange={(e) => setConfig({ ...iotdbConfig, port: Number(e.target.value) })}
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">用户</label>
        <input
          type="text"
          value={iotdbConfig.user}
          onChange={(e) => setConfig({ ...iotdbConfig, user: e.target.value })}
          required
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">密码</label>
        <input
          type="password"
          value={iotdbConfig.password}
          onChange={(e) => setConfig({ ...iotdbConfig, password: e.target.value })}
          placeholder="使用环境变量 ${IOTDB_PASS}"
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">时区</label>
        <input
          type="text"
          value={iotdbConfig.zone_id}
          onChange={(e) => setConfig({ ...iotdbConfig, zone_id: e.target.value })}
          className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
        />
      </div>
      <div className="flex items-center space-x-4">
        <button
          type="button"
          onClick={() => testMutation.mutate()}
          disabled={testing}
          className="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 disabled:opacity-50 focus-visible-ring"
        >
          {testing ? '测试中…' : '测试连接'}
        </button>
        {testResult && (
          <span className={testResult.success ? 'text-green-600' : 'text-red-600'}>
            {testResult.success 
              ? `✓ ${testResult.message || '连接成功'}` 
              : `✗ ${testResult.error || testResult.message || '连接失败'}`}
          </span>
        )}
      </div>
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
          className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
        >
          保存
        </button>
      </div>
    </form>
  )
}


