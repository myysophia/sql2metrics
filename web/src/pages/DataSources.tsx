import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { Config, MySQLConfig, IoTDBConfig, RedisConfig } from '../types/config'
import DataSourceForm from '../components/DataSourceForm'

export default function DataSources() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMySQL, setEditingMySQL] = useState<string | null>(null)
  const [editingIoTDB, setEditingIoTDB] = useState(false)
  const [editingRedis, setEditingRedis] = useState<string | null>(null)

  const updateConfigMutation = useMutation({
    mutationFn: (newConfig: Config) => api.updateConfig(newConfig),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingMySQL(null)
      setEditingIoTDB(false)
      setEditingRedis(null)
    },
  })

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const mysqlConnections = config.mysql_connections ?? {}
  const redisConnections = config.redis_connections ?? {}

  const handleSaveMySQL = (name: string, mysqlConfig: MySQLConfig) => {
    const newConfig: Config = {
      ...config,
      mysql_connections: {
        ...mysqlConnections,
        [name]: mysqlConfig,
      },
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleSaveRedis = (name: string, redisConfig: RedisConfig) => {
    const newConfig: Config = {
      ...config,
      redis_connections: {
        ...redisConnections,
        [name]: redisConfig,
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

    const newConnections = { ...mysqlConnections }
    delete newConnections[name]
    const newConfig: Config = {
      ...config,
      mysql_connections: newConnections,
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleDeleteRedis = (name: string) => {
    if (!confirm(`确定要删除 Redis 连接 "${name}" 吗？`)) return

    const newConnections = { ...redisConnections }
    delete newConnections[name]
    const newConfig: Config = {
      ...config,
      redis_connections: newConnections,
    }
    updateConfigMutation.mutate(newConfig)
  }

  const mysqlEntries = Object.entries(mysqlConnections) as [string, MySQLConfig][]
  const redisEntries = Object.entries(redisConnections) as [string, RedisConfig][]
  const defaultMySQL: MySQLConfig = { host: '', port: 3306, user: '', password: '', database: '', params: {} }
  const defaultRedis: RedisConfig = { mode: 'standalone', addr: '', db: 0, enable_tls: false, skip_tls_verify: false }

  let mysqlList: [string, MySQLConfig][] = mysqlEntries
  if (editingMySQL && !mysqlEntries.some(([name]) => name === editingMySQL)) {
    mysqlList = [...mysqlEntries, [editingMySQL, defaultMySQL]]
  }

  let redisList: [string, RedisConfig][] = redisEntries
  if (editingRedis && !redisEntries.some(([name]) => name === editingRedis)) {
    redisList = [...redisEntries, [editingRedis, defaultRedis]]
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
          {mysqlList.map(([name, mysqlConfig]) => (
            <div key={name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex justify-between items-center mb-2">
                <h4 className="font-medium">{name}</h4>
                <div className="space-x-2">
                  <button onClick={() => setEditingMySQL(name)} className="text-primary-600 hover:text-primary-700 focus-visible-ring">
                    编辑
                  </button>
                  {name !== 'default' && (
                    <button onClick={() => handleDeleteMySQL(name)} className="text-red-600 hover:text-red-700 focus-visible-ring">
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
                  <div>
                    {mysqlConfig.host}:{mysqlConfig.port}
                  </div>
                  <div>数据库: {mysqlConfig.database}</div>
                  <div>用户: {mysqlConfig.user}</div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Redis 连接 */}
      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-lg font-semibold">Redis 连接</h3>
          <button
            onClick={() => {
              const name = prompt('请输入连接名称:')
              if (name) setEditingRedis(name)
            }}
            className="px-4 py-2 bg-primary-600 text-white rounded hover:bg-primary-700 focus-visible-ring"
          >
            添加连接
          </button>
        </div>

        <div className="space-y-4">
          {redisList.map(([name, redisConfig]) => (
            <div key={name} className="border border-gray-200 rounded-lg p-4">
              <div className="flex justify-between items-center mb-2">
                <h4 className="font-medium">{name}</h4>
                <div className="space-x-2">
                  <button onClick={() => setEditingRedis(name)} className="text-primary-600 hover:text-primary-700 focus-visible-ring">
                    编辑
                  </button>
                  {name !== 'default' && (
                    <button onClick={() => handleDeleteRedis(name)} className="text-red-600 hover:text-red-700 focus-visible-ring">
                      删除
                    </button>
                  )}
                </div>
              </div>
              {editingRedis === name ? (
                <DataSourceForm
                  type="redis"
                  initialConfig={redisConfig}
                  onSave={(cfg) => handleSaveRedis(name, cfg as RedisConfig)}
                  onCancel={() => setEditingRedis(null)}
                />
              ) : (
                <div className="text-sm text-gray-600">
                  <div>{redisConfig.addr}</div>
                  <div>模式: {redisConfig.mode || 'standalone'}</div>
                  <div>DB: {redisConfig.db ?? 0}</div>
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
            <div>
              {config.iotdb.host}:{config.iotdb.port}
            </div>
            <div>用户: {config.iotdb.user}</div>
            <div>时区: {config.iotdb.zone_id}</div>
          </div>
        )}
      </div>
    </div>
  )
}
