import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../api/client'
import type { MySQLConfig, IoTDBConfig, RedisConfig } from '../types/config'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Loader2, Check, X } from 'lucide-react'

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
        <div className="space-y-2">
          <Label>主机</Label>
          <Input
            type="text"
            value={mysqlConfig.host}
            onChange={(e) => setConfig({ ...mysqlConfig, host: e.target.value })}
            required
          />
        </div>
        <div className="space-y-2">
          <Label>端口</Label>
          <Input
            type="number"
            value={mysqlConfig.port}
            onChange={(e) => setConfig({ ...mysqlConfig, port: Number(e.target.value) })}
            required
          />
        </div>
        <div className="space-y-2">
          <Label>用户</Label>
          <Input
            type="text"
            value={mysqlConfig.user}
            onChange={(e) => setConfig({ ...mysqlConfig, user: e.target.value })}
            required
          />
        </div>
        <div className="space-y-2">
          <Label>密码</Label>
          <Input
            type="password"
            value={mysqlConfig.password}
            onChange={(e) => setConfig({ ...mysqlConfig, password: e.target.value })}
            placeholder="使用环境变量 ${MYSQL_PASS}"
          />
        </div>
        <div className="space-y-2">
          <Label>数据库</Label>
          <Input
            type="text"
            value={mysqlConfig.database}
            onChange={(e) => setConfig({ ...mysqlConfig, database: e.target.value })}
            required
          />
        </div>

        <div className="flex items-center space-x-4">
          <Button
            type="button"
            variant="secondary"
            onClick={() => testMutation.mutate()}
            disabled={testing}
          >
            {testing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            {testing ? '测试中…' : '测试连接'}
          </Button>
          {testResult && (
            <span className={`flex items-center text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
              {testResult.success ? <Check className="mr-1 h-4 w-4" /> : <X className="mr-1 h-4 w-4" />}
              {testResult.success ? '连接成功' : testResult.message}
            </span>
          )}
        </div>

        <div className="flex justify-end space-x-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button type="submit">
            保存
          </Button>
        </div>
      </form>
    )
  }

  if (type === 'redis') {
    const redisConfig = config as RedisConfig
    return (
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label>模式</Label>
          <Select
            value={redisConfig.mode || 'standalone'}
            onValueChange={(value) => setConfig({ ...redisConfig, mode: value as RedisConfig['mode'] })}
          >
            <SelectTrigger>
              <SelectValue placeholder="选择模式" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="standalone">Standalone</SelectItem>
              <SelectItem value="sentinel" disabled>Sentinel（暂不支持）</SelectItem>
              <SelectItem value="cluster" disabled>Cluster（暂不支持）</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>地址</Label>
          <Input
            type="text"
            value={redisConfig.addr}
            onChange={(e) => setConfig({ ...redisConfig, addr: e.target.value })}
            required
            placeholder="127.0.0.1:6379"
          />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>用户名</Label>
            <Input
              type="text"
              value={redisConfig.username || ''}
              onChange={(e) => setConfig({ ...redisConfig, username: e.target.value || undefined })}
            />
          </div>
          <div className="space-y-2">
            <Label>密码</Label>
            <Input
              type="password"
              value={redisConfig.password || ''}
              onChange={(e) => setConfig({ ...redisConfig, password: e.target.value || undefined })}
              placeholder="使用环境变量 ${REDIS_PASS}"
            />
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>DB</Label>
            <Input
              type="number"
              value={redisConfig.db ?? 0}
              min={0}
              onChange={(e) => setConfig({ ...redisConfig, db: Number(e.target.value) })}
            />
          </div>
          <div className="flex items-center space-x-2 mt-8">
            <Checkbox
              id="redis-tls"
              checked={!!redisConfig.enable_tls}
              onCheckedChange={(checked) => setConfig({ ...redisConfig, enable_tls: checked as boolean })}
            />
            <Label htmlFor="redis-tls" className="font-normal">
              启用 TLS
            </Label>
          </div>
        </div>
        {redisConfig.enable_tls && (
          <div className="flex items-center space-x-2">
            <Checkbox
              id="redis-skip-verify"
              checked={!!redisConfig.skip_tls_verify}
              onCheckedChange={(checked) => setConfig({ ...redisConfig, skip_tls_verify: checked as boolean })}
            />
            <Label htmlFor="redis-skip-verify" className="font-normal">
              跳过证书验证（仅测试环境）
            </Label>
          </div>
        )}

        <div className="flex items-center space-x-4">
          <Button
            type="button"
            variant="secondary"
            onClick={() => testMutation.mutate()}
            disabled={testing}
          >
            {testing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            {testing ? '测试中…' : '测试连接'}
          </Button>
          {testResult && (
            <span className={`flex items-center text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
              {testResult.success ? <Check className="mr-1 h-4 w-4" /> : <X className="mr-1 h-4 w-4" />}
              {testResult.success ? '连接成功' : testResult.message}
            </span>
          )}
        </div>

        <div className="flex justify-end space-x-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            取消
          </Button>
          <Button type="submit">
            保存
          </Button>
        </div>
      </form>
    )
  }

  const iotdbConfig = config as IoTDBConfig
  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label>主机</Label>
        <Input
          type="text"
          value={iotdbConfig.host}
          onChange={(e) => setConfig({ ...iotdbConfig, host: e.target.value })}
          required
        />
      </div>
      <div className="space-y-2">
        <Label>端口</Label>
        <Input
          type="number"
          value={iotdbConfig.port}
          onChange={(e) => setConfig({ ...iotdbConfig, port: Number(e.target.value) })}
          required
        />
      </div>
      <div className="space-y-2">
        <Label>用户</Label>
        <Input
          type="text"
          value={iotdbConfig.user}
          onChange={(e) => setConfig({ ...iotdbConfig, user: e.target.value })}
          required
        />
      </div>
      <div className="space-y-2">
        <Label>密码</Label>
        <Input
          type="password"
          value={iotdbConfig.password}
          onChange={(e) => setConfig({ ...iotdbConfig, password: e.target.value })}
          placeholder="使用环境变量 ${IOTDB_PASS}"
        />
      </div>
      <div className="space-y-2">
        <Label>时区</Label>
        <Input
          type="text"
          value={iotdbConfig.zone_id}
          onChange={(e) => setConfig({ ...iotdbConfig, zone_id: e.target.value })}
        />
      </div>

      <div className="flex items-center space-x-4">
        <Button
          type="button"
          variant="secondary"
          onClick={() => testMutation.mutate()}
          disabled={testing}
        >
          {testing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          {testing ? '测试中…' : '测试连接'}
        </Button>
        {testResult && (
          <span className={`flex items-center text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
            {testResult.success ? <Check className="mr-1 h-4 w-4" /> : <X className="mr-1 h-4 w-4" />}
            {testResult.success ? '连接成功' : testResult.message}
          </span>
        )}
      </div>

      <div className="flex justify-end space-x-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button type="submit">
          保存
        </Button>
      </div>
    </form>
  )
}
