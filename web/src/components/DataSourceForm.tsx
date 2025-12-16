import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import type { MySQLConfig, IoTDBConfig, RedisConfig } from '../types/config'

interface DataSourceFormProps {
  type: 'mysql' | 'iotdb' | 'redis'
  initialConfig: MySQLConfig | IoTDBConfig | RedisConfig
  onSave: (config: MySQLConfig | IoTDBConfig | RedisConfig) => void
  onCancel: () => void
}

export default function DataSourceForm({ type, initialConfig, onSave, onCancel }: DataSourceFormProps) {
  const [config, setConfig] = useState(initialConfig)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ success: boolean; message?: string } | null>(null)

  const testMutation = useMutation({
    mutationFn: async () => {
      if (type === 'mysql') return api.testMySQL(config as MySQLConfig)
      if (type === 'iotdb') return api.testIoTDB(config as IoTDBConfig)
      return api.testRedis(config as RedisConfig)
    },
    onMutate: () => {
      setTesting(true)
      setTestResult(null)
    },
    onSuccess: (result) => {
      if (!result.success) {
        setTestResult({ success: false, message: result.error || result.message || '连接失败' })
      } else {
        setTestResult({ success: true, message: result.message })
      }
      setTesting(false)
    },
    onError: (error: Error) => {
      setTestResult({ success: false, message: error.message })
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
              {testResult.success ? '✓ 连接成功' : `✗ ${testResult.message}`}
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
          <button type="submit" className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring">
            保存
          </button>
        </div>
      </form>
    )
  }

  if (type === 'redis') {
    const redisConfig = config as RedisConfig
    return (
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">模式</label>
          <select
            value={redisConfig.mode || 'standalone'}
            onChange={(e) => setConfig({ ...redisConfig, mode: e.target.value as RedisConfig['mode'] })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          >
            <option value="standalone">Standalone</option>
            <option value="sentinel" disabled>
              Sentinel（暂不支持）
            </option>
            <option value="cluster" disabled>
              Cluster（暂不支持）
            </option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">地址</label>
          <input
            type="text"
            value={redisConfig.addr}
            onChange={(e) => setConfig({ ...redisConfig, addr: e.target.value })}
            required
            placeholder="127.0.0.1:6379"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
          />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">用户名</label>
            <input
              type="text"
              value={redisConfig.username || ''}
              onChange={(e) => setConfig({ ...redisConfig, username: e.target.value || undefined })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">密码</label>
            <input
              type="password"
              value={redisConfig.password || ''}
              onChange={(e) => setConfig({ ...redisConfig, password: e.target.value || undefined })}
              placeholder="使用环境变量 ${REDIS_PASS}"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            />
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">DB</label>
            <input
              type="number"
              value={redisConfig.db ?? 0}
              min={0}
              onChange={(e) => setConfig({ ...redisConfig, db: Number(e.target.value) })}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus-visible-ring"
            />
          </div>
          <div className="flex items-center space-x-2 mt-6">
            <input
              id="redis-tls"
              type="checkbox"
              checked={!!redisConfig.enable_tls}
              onChange={(e) => setConfig({ ...redisConfig, enable_tls: e.target.checked })}
              className="h-4 w-4 text-primary-600 border-gray-300 rounded focus-visible-ring"
            />
            <label htmlFor="redis-tls" className="text-sm text-gray-700">
              启用 TLS
            </label>
          </div>
        </div>
        {redisConfig.enable_tls && (
          <div className="flex items-center space-x-2">
            <input
              id="redis-skip-verify"
              type="checkbox"
              checked={!!redisConfig.skip_tls_verify}
              onChange={(e) => setConfig({ ...redisConfig, skip_tls_verify: e.target.checked })}
              className="h-4 w-4 text-primary-600 border-gray-300 rounded focus-visible-ring"
            />
            <label htmlFor="redis-skip-verify" className="text-sm text-gray-700">
              跳过证书验证（仅测试环境）
            </label>
          </div>
        )}

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
              {testResult.success ? '✓ 连接成功' : `✗ ${testResult.message}`}
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
          <button type="submit" className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring">
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
            {testResult.success ? '✓ 连接成功' : `✗ ${testResult.message}`}
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
        <button type="submit" className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring">
          保存
        </button>
      </div>
    </form>
  )
}
