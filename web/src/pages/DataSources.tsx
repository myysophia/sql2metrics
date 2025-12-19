import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { MySQLConfig, IoTDBConfig, RedisConfig, RestAPIConfig } from '../types/config'
import DataSourceForm from '../components/DataSourceForm'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Plus, Edit2, Trash2, Database, Server, Globe } from 'lucide-react'
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

export default function DataSources() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMySQL, setEditingMySQL] = useState<string | null>(null)
  const [editingIoTDB, setEditingIoTDB] = useState(false)
  const [editingRedis, setEditingRedis] = useState<string | null>(null)
  const [editingRestAPI, setEditingRestAPI] = useState<string | null>(null)

  const [isAddMySQLDialogOpen, setIsAddMySQLDialogOpen] = useState(false)
  const [isAddRedisDialogOpen, setIsAddRedisDialogOpen] = useState(false)
  const [isAddRestAPIDialogOpen, setIsAddRestAPIDialogOpen] = useState(false)

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const mysqlConnections = config.mysql_connections ?? {}
  const redisConnections = config.redis_connections ?? {}
  const restapiConnections = config.restapi_connections ?? {}

  const handleSaveMySQL = async (name: string, mysqlConfig: MySQLConfig) => {
    try {
      await api.updateMySQLConnection(name, mysqlConfig)
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingMySQL(null)
    } catch (error) {
      console.error('保存 MySQL 连接失败:', error)
      throw error
    }
  }

  const handleSaveRedis = async (name: string, redisConfig: RedisConfig) => {
    try {
      await api.updateRedisConnection(name, redisConfig)
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingRedis(null)
    } catch (error) {
      console.error('保存 Redis 连接失败:', error)
      throw error
    }
  }

  const handleSaveIoTDB = async (iotdbConfig: IoTDBConfig) => {
    try {
      await api.updateIoTDB(iotdbConfig)
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingIoTDB(false)
    } catch (error) {
      console.error('保存 IoTDB 配置失败:', error)
      throw error
    }
  }

  const handleDeleteMySQL = async (name: string) => {
    try {
      await api.deleteMySQLConnection(name)
      queryClient.invalidateQueries({ queryKey: ['config'] })
    } catch (error) {
      console.error('删除 MySQL 连接失败:', error)
      throw error
    }
  }

  const handleDeleteRedis = async (name: string) => {
    try {
      await api.deleteRedisConnection(name)
      queryClient.invalidateQueries({ queryKey: ['config'] })
    } catch (error) {
      console.error('删除 Redis 连接失败:', error)
      throw error
    }
  }

  const handleSaveRestAPI = async (name: string, restapiConfig: RestAPIConfig) => {
    try {
      await api.updateRestAPIConnection(name, restapiConfig)
      queryClient.invalidateQueries({ queryKey: ['config'] })
      setEditingRestAPI(null)
    } catch (error) {
      console.error('保存 RestAPI 连接失败:', error)
      throw error
    }
  }

  const handleDeleteRestAPI = async (name: string) => {
    try {
      await api.deleteRestAPIConnection(name)
      queryClient.invalidateQueries({ queryKey: ['config'] })
    } catch (error) {
      console.error('删除 RestAPI 连接失败:', error)
      throw error
    }
  }

  const mysqlEntries = Object.entries(mysqlConnections) as [string, MySQLConfig][]
  const redisEntries = Object.entries(redisConnections) as [string, RedisConfig][]
  const restapiEntries = Object.entries(restapiConnections) as [string, RestAPIConfig][]
  const defaultMySQL: MySQLConfig = { host: '', port: 3306, user: '', password: '', database: '', params: {} }
  const defaultRedis: RedisConfig = { mode: 'standalone', addr: '', db: 0, enable_tls: false, skip_tls_verify: false }
  const defaultRestAPI: RestAPIConfig = { base_url: '', timeout: '30s' }

  let mysqlList: [string, MySQLConfig][] = mysqlEntries
  if (editingMySQL && !mysqlEntries.some(([name]) => name === editingMySQL)) {
    mysqlList = [...mysqlEntries, [editingMySQL, defaultMySQL]]
  }

  let redisList: [string, RedisConfig][] = redisEntries
  if (editingRedis && !redisEntries.some(([name]) => name === editingRedis)) {
    redisList = [...redisEntries, [editingRedis, defaultRedis]]
  }

  let restapiList: [string, RestAPIConfig][] = restapiEntries
  if (editingRestAPI && !restapiEntries.some(([name]) => name === editingRestAPI)) {
    restapiList = [...restapiEntries, [editingRestAPI, defaultRestAPI]]
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
                        <div className="flex justify-between items-center overflow-hidden">
                          <span className="shrink-0 mr-2">地址:</span>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="font-medium text-foreground truncate cursor-help">
                                  {mysqlConfig.host}:{mysqlConfig.port}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{mysqlConfig.host}:{mysqlConfig.port}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
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
                        <div className="flex justify-between items-center overflow-hidden">
                          <span className="shrink-0 mr-2">地址:</span>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="font-medium text-foreground truncate cursor-help">
                                  {redisConfig.addr}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{redisConfig.addr}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
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
              <div className="flex items-center gap-2 overflow-hidden">
                <span className="w-12 shrink-0">地址:</span>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="font-medium text-foreground truncate cursor-help">
                        {config.iotdb.host}:{config.iotdb.port}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{config.iotdb.host}:{config.iotdb.port}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
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

      {/* RestAPI 连接 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <div className="space-y-1">
            <CardTitle className="text-xl flex items-center gap-2">
              <Globe className="h-5 w-5" /> RestAPI 连接
            </CardTitle>
            <CardDescription>管理 RESTful API 数据源配置</CardDescription>
          </div>
          <Button
            onClick={() => setIsAddRestAPIDialogOpen(true)}
            size="sm"
          >
            <Plus className="mr-2 h-4 w-4" /> 添加连接
          </Button>
        </CardHeader>
        <CardContent className="space-y-4 pt-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {restapiList.map(([name, restapiConfig]) => (
              <Card key={name} className="overflow-hidden">
                {editingRestAPI === name ? (
                  <div className="p-4">
                    <div className="flex justify-between items-center mb-4">
                      <h4 className="font-medium">{name}</h4>
                    </div>
                    <DataSourceForm
                      type="restapi"
                      initialConfig={restapiConfig}
                      onSave={(cfg) => handleSaveRestAPI(name, cfg as RestAPIConfig)}
                      onCancel={() => setEditingRestAPI(null)}
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
                          onClick={() => setEditingRestAPI(name)}
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
                                <AlertDialogTitle>确定要删除 RestAPI 连接 "{name}" 吗？</AlertDialogTitle>
                                <AlertDialogDescription>
                                  此操作不可撤销。这将永久删除该连接配置，可能会影响使用此连接的指标。
                                </AlertDialogDescription>
                              </AlertDialogHeader>
                              <AlertDialogFooter>
                                <AlertDialogCancel>取消</AlertDialogCancel>
                                <AlertDialogAction
                                  onClick={() => handleDeleteRestAPI(name)}
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
                        <div className="flex justify-between items-center overflow-hidden">
                          <span className="shrink-0 mr-2">Base URL:</span>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="font-medium text-foreground truncate cursor-help">
                                  {restapiConfig.base_url || '未配置'}
                                </span>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{restapiConfig.base_url || '未配置'}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                        <div className="flex justify-between">
                          <span>超时:</span>
                          <span className="font-medium text-foreground">{restapiConfig.timeout || '30s'}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>TLS 跳过验证:</span>
                          <span className="font-medium text-foreground">{restapiConfig.tls?.skip_verify ? '是' : '否'}</span>
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

      <AddConnectionDialog
        open={isAddRestAPIDialogOpen}
        onOpenChange={setIsAddRestAPIDialogOpen}
        onConfirm={(name) => {
          setEditingRestAPI(name)
        }}
        title="添加 RestAPI 连接"
        description="请输入新的 RestAPI 连接名称，添加后可进行详细配置。"
      />
    </div>
  )
}
