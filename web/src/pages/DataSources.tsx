import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { Config, MySQLConfig, IoTDBConfig, HTTPAPIConfig } from '../types/config'
import DataSourceForm from '../components/DataSourceForm'

export default function DataSources() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMySQL, setEditingMySQL] = useState<string | null>(null)
  const [editingIoTDB, setEditingIoTDB] = useState(false)
  const [editingHTTPAPI, setEditingHTTPAPI] = useState<string | null>(null)

  const updateConfigMutation = useMutation({
    mutationFn: (newConfig: Config) => api.updateConfig(newConfig),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingMySQL(null)
      setEditingIoTDB(false)
      setEditingHTTPAPI(null)
    },
  })

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const handleSaveMySQL = (name: string, mysqlConfig: MySQLConfig) => {
    const newConfig: Config = {
      ...config,
      mysql_connections: {
        ...config.mysql_connections,
        [name]: mysqlConfig,
      },
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleSaveIoTDB = (iotdbConfig: IoTDBConfig) => {
    const newConfig: Config = {
      ...config,
      iotdb: iotdbConfig,
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleDeleteMySQL = (name: string) => {
    if (!confirm(`确定要删除 MySQL 连接 "${name}" 吗？`)) return

    const newConnections = { ...config.mysql_connections }
    delete newConnections[name]
    const newConfig: Config = {
      ...config,
      mysql_connections: newConnections,
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleSaveHTTPAPI = (name: string, httpapiConfig: HTTPAPIConfig) => {
    const newConfig: Config = {
      ...config,
      http_api_connections: {
        ...config.http_api_connections,
        [name]: httpapiConfig,
      },
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleDeleteHTTPAPI = (name: string) => {
    if (!confirm(`确定要删除 HTTP API 连接 "${name}" 吗？`)) return

    const newConnections = { ...(config.http_api_connections || {}) }
    delete newConnections[name]
    const newConfig: Config = {
      ...config,
      http_api_connections: newConnections,
    }
    updateConfigMutation.mutate(newConfig)
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">数据源管理</h2>

      {/* MySQL 连接 */}
      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold">MySQL 连接</h3>
          <button
            onClick={() => {
              const name = prompt('请输入连接名称:')
              if (name) setEditingMySQL(name)
            }}
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
          >
            添加连接
          </button>
        </div>

        <div className="space-y-4">
          {Object.entries(config.mysql_connections || {}).map(([name, mysqlConfig]) => (
            <div key={name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex justify-between items-center mb-2">
                <h4 className="font-medium">{name}</h4>
                <div className="space-x-2">
                  <button
                    onClick={() => setEditingMySQL(name)}
                    className="text-primary-600 hover:text-primary-700 focus-visible-ring"
                  >
                    编辑
                  </button>
                  {name !== 'default' && (
                    <button
                      onClick={() => handleDeleteMySQL(name)}
                      className="text-red-600 hover:text-red-700 focus-visible-ring"
                    >
                      删除
                    </button>
                  )}
                </div>
              </div>
              {editingMySQL === name ? (
                <DataSourceForm
                  type="mysql"
                  initialConfig={mysqlConfig}
                  onSave={(cfg) => handleSaveMySQL(name, cfg as MySQLConfig)}
                  onCancel={() => setEditingMySQL(null)}
                />
              ) : (
                <div className="text-sm text-gray-600">
                  <div>{mysqlConfig.host}:{mysqlConfig.port}</div>
                  <div>数据库: {mysqlConfig.database}</div>
                  <div>用户: {mysqlConfig.user}</div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* IoTDB 连接 */}
      <div className="bg-white rounded-lg shadow p-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold">IoTDB 连接</h3>
          {!editingIoTDB && (
            <button
              onClick={() => setEditingIoTDB(true)}
              className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
            >
              编辑
            </button>
          )}
        </div>

        {editingIoTDB ? (
          <DataSourceForm
            type="iotdb"
            initialConfig={config.iotdb}
            onSave={(cfg) => handleSaveIoTDB(cfg as IoTDBConfig)}
            onCancel={() => setEditingIoTDB(false)}
          />
        ) : (
          <div className="text-sm text-gray-600">
            <div>{config.iotdb.host}:{config.iotdb.port}</div>
            <div>用户: {config.iotdb.user}</div>
            <div>时区: {config.iotdb.zone_id}</div>
          </div>
        )}
      </div>

      {/* HTTP API 连接 */}
      <div className="bg-white rounded-lg shadow p-6 mt-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold">HTTP API 连接</h3>
          <button
            onClick={() => {
              const name = prompt('请输入连接名称:')
              if (name) setEditingHTTPAPI(name)
            }}
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
          >
            添加连接
          </button>
        </div>

        <div className="space-y-4">
          {Object.entries(config.http_api_connections || {}).map(([name, httpapiConfig]) => (
            <div key={name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex justify-between items-center mb-2">
                <h4 className="font-medium">{name}</h4>
                <div className="space-x-2">
                  <button
                    onClick={() => setEditingHTTPAPI(name)}
                    className="text-primary-600 hover:text-primary-700 focus-visible-ring"
                  >
                    编辑
                  </button>
                  {name !== 'default' && (
                    <button
                      onClick={() => handleDeleteHTTPAPI(name)}
                      className="text-red-600 hover:text-red-700 focus-visible-ring"
                    >
                      删除
                    </button>
                  )}
                </div>
              </div>
              {editingHTTPAPI === name ? (
                <DataSourceForm
                  type="http_api"
                  initialConfig={httpapiConfig}
                  onSave={(cfg) => handleSaveHTTPAPI(name, cfg as HTTPAPIConfig)}
                  onCancel={() => setEditingHTTPAPI(null)}
                />
              ) : (
                <div className="text-sm text-gray-600">
                  <div>URL: {httpapiConfig.url}</div>
                  <div>方法: {httpapiConfig.method || 'GET'}</div>
                  <div>超时: {httpapiConfig.timeout || 10}秒</div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}


