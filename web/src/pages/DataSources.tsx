import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { Config, MySQLConfig, IoTDBConfig, RedisConfig } from '../types/config'
import DataSourceForm from '../components/DataSourceForm'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Plus, Edit2, Trash2, Database, Server } from 'lucide-react'
import { AddConnectionDialog } from '@/components/AddConnectionDialog'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"

export default function DataSources() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMySQL, setEditingMySQL] = useState<string | null>(null)
  const [editingIoTDB, setEditingIoTDB] = useState(false)
  const [editingRedis, setEditingRedis] = useState<string | null>(null)

  const [isAddMySQLDialogOpen, setIsAddMySQLDialogOpen] = useState(false)
  const [isAddRedisDialogOpen, setIsAddRedisDialogOpen] = useState(false)

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

    const newConnections = { ...mysqlConnections }
    delete newConnections[name]
    const newConfig: Config = {
      ...config,
      mysql_connections: newConnections,
    }
    updateConfigMutation.mutate(newConfig)
  }

  const handleDeleteRedis = (name: string) => {

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
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">数据源管理</h2>
      </div>

      {/* MySQL 连接 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle className="text-xl flex items-center gap-2">
              <Database className="h-5 w-5" /> MySQL 连接
            </CardTitle>
            <CardDescription>管理 MySQL 数据库连接配置</CardDescription>
          </div>
          <Button
            onClick={() => setIsAddMySQLDialogOpen(true)}
            size="sm"
          >
            <Plus className="mr-2 h-4 w-4" /> 添加连接
          </Button>
        </CardHeader>
        <CardContent className="space-y-4 pt-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {mysqlList.map(([name, mysqlConfig]) => (
              <Card key={name} className="overflow-hidden">
                {editingMySQL === name ? (
                  <div className="p-4">
                    <div className="flex justify-between items-center mb-4">
                      <h4 className="font-medium">{name}</h4>
                    </div>
                    <DataSourceForm
                      type="mysql"
                      initialConfig={mysqlConfig}
                      onSave={(cfg) => handleSaveMySQL(name, cfg as MySQLConfig)}
                      onCancel={() => setEditingMySQL(null)}
                    />
                  </div>
                ) : (
                  <>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                      <CardTitle className="text-base font-medium">{name}</CardTitle>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setEditingMySQL(name)}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        {name !== 'default' && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="text-destructive hover:text-destructive"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>确定要删除 MySQL 连接 "{name}" 吗？</AlertDialogTitle>
                                <AlertDialogDescription>
                                  此操作不可撤销。这将永久删除该连接配置，可能会影响使用此连接的指标。
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>取消</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDeleteMySQL(name)}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  删除
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="text-sm text-muted-foreground space-y-1">
                        <div className="flex justify-between">
                          <span>地址:</span>
                          <span className="font-medium text-foreground">{mysqlConfig.host}:{mysqlConfig.port}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>数据库:</span>
                          <span className="font-medium text-foreground">{mysqlConfig.database}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>用户:</span>
                          <span className="font-medium text-foreground">{mysqlConfig.user}</span>
                        </div>
                      </div>
                    </CardContent>
                  </>
                )}
              </Card>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Redis 连接 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle className="text-xl flex items-center gap-2">
              <Server className="h-5 w-5" /> Redis 连接
            </CardTitle>
            <CardDescription>管理 Redis 数据库连接配置</CardDescription>
          </div>
          <Button
            onClick={() => setIsAddRedisDialogOpen(true)}
            size="sm"
          >
            <Plus className="mr-2 h-4 w-4" /> 添加连接
          </Button>
        </CardHeader>
        <CardContent className="space-y-4 pt-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {redisList.map(([name, redisConfig]) => (
              <Card key={name} className="overflow-hidden">
                {editingRedis === name ? (
                  <div className="p-4">
                    <div className="flex justify-between items-center mb-4">
                      <h4 className="font-medium">{name}</h4>
                    </div>
                    <DataSourceForm
                      type="redis"
                      initialConfig={redisConfig}
                      onSave={(cfg) => handleSaveRedis(name, cfg as RedisConfig)}
                      onCancel={() => setEditingRedis(null)}
                    />
                  </div>
                ) : (
                  <>
                    <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                      <CardTitle className="text-base font-medium">{name}</CardTitle>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setEditingRedis(name)}
                        >
                          <Edit2 className="h-4 w-4" />
                        </Button>
                        {name !== 'default' && (
                          <AlertDialog>
                            <AlertDialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="text-destructive hover:text-destructive"
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </AlertDialogTrigger>
                            <AlertDialogContent>
                              <AlertDialogHeader>
                                <AlertDialogTitle>确定要删除 Redis 连接 "{name}" 吗？</AlertDialogTitle>
                                <AlertDialogDescription>
                                  此操作不可撤销。这将永久删除该连接配置，可能会影响使用此连接的指标。
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>取消</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDeleteRedis(name)}
                                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                                >
                                  删除
                                </AlertDialogAction>
                              </AlertDialogFooter>
                            </AlertDialogContent>
                          </AlertDialog>
                        )}
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="text-sm text-muted-foreground space-y-1">
                        <div className="flex justify-between">
                          <span>地址:</span>
                          <span className="font-medium text-foreground">{redisConfig.addr}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>模式:</span>
                          <span className="font-medium text-foreground">{redisConfig.mode || 'standalone'}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>DB:</span>
                          <span className="font-medium text-foreground">{redisConfig.db ?? 0}</span>
                        </div>
                      </div>
                    </CardContent>
                  </>
                )}
              </Card>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* IoTDB 连接 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle className="text-xl flex items-center gap-2">
              <Database className="h-5 w-5" /> IoTDB 连接
            </CardTitle>
            <CardDescription>管理 IoTDB 时序数据库配置</CardDescription>
          </div>
          {!editingIoTDB && (
            <Button
              onClick={() => setEditingIoTDB(true)}
              size="sm"
              variant="outline"
            >
              <Edit2 className="mr-2 h-4 w-4" /> 编辑配置
            </Button>
          )}
        </CardHeader>
        <CardContent className="pt-4">
          {editingIoTDB ? (
            <div className="max-w-xl">
              <DataSourceForm
                type="iotdb"
                initialConfig={config.iotdb}
                onSave={(cfg) => handleSaveIoTDB(cfg as IoTDBConfig)}
                onCancel={() => setEditingIoTDB(false)}
              />
            </div>
          ) : (
            <div className="text-sm text-muted-foreground space-y-1">
              <div className="flex items-center gap-2">
                <span className="w-12">地址:</span>
                <span className="font-medium text-foreground">{config.iotdb.host}:{config.iotdb.port}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="w-12">用户:</span>
                <span className="font-medium text-foreground">{config.iotdb.user}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="w-12">时区:</span>
                <span className="font-medium text-foreground">{config.iotdb.zone_id}</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
      <AddConnectionDialog
        open={isAddMySQLDialogOpen}
        onOpenChange={setIsAddMySQLDialogOpen}
        onConfirm={(name) => {
          setEditingMySQL(name)
        }}
        title="添加 MySQL 连接"
        description="请输入新的 MySQL 连接名称，添加后可进行详细配置。"
      />

      <AddConnectionDialog
        open={isAddRedisDialogOpen}
        onOpenChange={setIsAddRedisDialogOpen}
        onConfirm={(name) => {
          setEditingRedis(name)
        }}
        title="添加 Redis 连接"
        description="请输入新的 Redis 连接名称，添加后可进行详细配置。"
      />
    </div>
  )
}
