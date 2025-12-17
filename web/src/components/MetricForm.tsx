import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import Editor from '@monaco-editor/react'
import { api } from '../api/client'
import type { Config, MetricSpec } from '../types/config'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Loader2, Plus, Trash2, HelpCircle } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { useToast } from "@/hooks/use-toast"

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
  const [showSaveConfirm, setShowSaveConfirm] = useState(false)

  const metricTypeTips: Record<MetricSpec['type'], string> = {
    gauge: 'Gauge：表示某一时刻的数值快照，可上可下（例如温度、队列长度）。',
    counter: 'Counter：只增不减的累计值（例如请求总数、错误总数）。',
    histogram: 'Histogram：按桶统计分布，适合延迟/大小等需要分布的指标。',
    summary: 'Summary：在客户端计算分位数，适合看 P99 等分位但聚合能力有限。',
  }

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

  const { toast } = useToast()

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (initialMetric.name && initialMetric.name === metric.name) {
        return api.updateMetric(metric.name, metric)
      }
      return api.createMetric(metric)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
      toast({
        title: "保存成功",
        description: `指标 ${metric.name} 已成功保存`,
      })
      onSave()
    },
    onError: (error: Error) => {
      toast({
        variant: "destructive",
        title: "保存失败",
        description: error.message || "未知错误，请检查网络或后端日志",
      })
      setShowSaveConfirm(false)
    }
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setShowSaveConfirm(true)
  }

  const handleConfirmSave = () => {
    saveMutation.mutate()
    setShowSaveConfirm(false)
  }

  const mysqlConnections = Object.keys(config.mysql_connections || {})
  const redisConnections = Object.keys(config.redis_connections || {})

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label>
            指标名称 <span className="text-red-500">*</span>
          </Label>
          <div className="space-y-1">
            <Input
              type="text"
              value={metric.name}
              onChange={(e) => setMetric({ ...metric, name: e.target.value })}
              required
              pattern="[a-zA-Z_:][a-zA-Z0-9_:]*"
              placeholder="energy_household_total"
            />
            <p className="text-xs text-muted-foreground">只能包含字母、数字、下划线和冒号</p>
          </div>
        </div>

        <div className="space-y-2">
          <Label className="flex items-center gap-1">
            指标类型 <span className="text-red-500">*</span>
            {/* Tooltip requires provider, assuming global or local. Using local for safety if not in Layout */}
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <HelpCircle className="h-4 w-4 text-muted-foreground cursor-help" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>{metricTypeTips[metric.type]}</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </Label>
          <Select
            value={metric.type}
            onValueChange={(value) => setMetric({ ...metric, type: value as MetricSpec['type'] })}
          >
            <SelectTrigger>
              <SelectValue placeholder="选择类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="gauge">Gauge</SelectItem>
              <SelectItem value="counter">Counter</SelectItem>
              <SelectItem value="histogram">Histogram</SelectItem>
              <SelectItem value="summary">Summary</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <Label>
          帮助信息 <span className="text-red-500">*</span>
        </Label>
        <Input
          type="text"
          value={metric.help}
          onChange={(e) => setMetric({ ...metric, help: e.target.value })}
          required
          placeholder="家庭用电量"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label>
            数据源 <span className="text-red-500">*</span>
          </Label>
          <Select
            value={metric.source}
            onValueChange={(value) => setMetric({ ...metric, source: value as MetricSpec['source'], connection: undefined, result_field: undefined })}
          >
            <SelectTrigger>
              <SelectValue placeholder="选择数据源" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="mysql">MySQL</SelectItem>
              <SelectItem value="iotdb">IoTDB</SelectItem>
              <SelectItem value="redis">Redis</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {metric.source === 'mysql' && mysqlConnections.length > 0 && (
          <div className="space-y-2">
            <Label>连接</Label>
            <Select
              value={metric.connection}
              onValueChange={(value) => setMetric({ ...metric, connection: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择连接" />
              </SelectTrigger>
              <SelectContent>
                {mysqlConnections.map((conn) => (
                  <SelectItem key={conn} value={conn}>
                    {conn}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {metric.source === 'redis' && redisConnections.length > 0 && (
          <div className="space-y-2">
            <Label>连接</Label>
            <Select
              value={metric.connection}
              onValueChange={(value) => setMetric({ ...metric, connection: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder="选择连接" />
              </SelectTrigger>
              <SelectContent>
                {redisConnections.map((conn) => (
                  <SelectItem key={conn} value={conn}>
                    {conn}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {metric.source === 'iotdb' && (
          <div className="space-y-2">
            <Label>结果字段</Label>
            <Input
              type="text"
              value={metric.result_field || ''}
              onChange={(e) => setMetric({ ...metric, result_field: e.target.value || undefined })}
              placeholder="留空则使用第一列"
            />
          </div>
        )}
      </div>

      <div className="space-y-2">
        <div className="flex justify-between items-center">
          <Label>
            查询/命令 <span className="text-red-500">*</span>
          </Label>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => previewMutation.mutate()}
            disabled={previewing || !metric.query}
          >
            {previewing ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            {previewing ? '预览中…' : '预览查询'}
          </Button>
        </div>
        <div className="border rounded-md overflow-hidden">
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
        {previewResult && (
          <div className={`mt-2 p-2 rounded text-sm ${previewResult.success ? 'bg-green-50 text-green-800' : 'bg-red-50 text-red-800'}`}>
            {previewResult.success ? (
              <div>
                查询结果: <span className="font-mono font-bold">{previewResult.value}</span>
              </div>
            ) : (
              <div>错误: {previewResult.error}</div>
            )}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Label>标签</Label>
        <div className="space-y-2">
          {Object.entries(metric.labels || {}).map(([key, value], index) => (
            <div key={index} className="flex space-x-2">
              <Input
                type="text"
                value={key}
                onChange={(e) => {
                  const newLabels = { ...metric.labels }
                  delete newLabels[key]
                  newLabels[e.target.value] = value
                  setMetric({ ...metric, labels: newLabels })
                }}
                placeholder="标签键"
                className="flex-1"
              />
              <Input
                type="text"
                value={value}
                onChange={(e) => {
                  setMetric({
                    ...metric,
                    labels: { ...metric.labels, [key]: e.target.value },
                  })
                }}
                placeholder="标签值"
                className="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => {
                  const newLabels = { ...metric.labels }
                  delete newLabels[key]
                  setMetric({ ...metric, labels: newLabels })
                }}
                className="text-destructive hover:text-destructive/90"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ))}
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              setMetric({
                ...metric,
                labels: { ...(metric.labels || {}), '': '' },
              })
            }}
            className="w-full border-dashed"
          >
            <Plus className="mr-2 h-4 w-4" /> 添加标签
          </Button>
        </div>
      </div>

      {metric.type === 'histogram' && (
        <div className="space-y-2">
          <Label>Buckets</Label>
          <Input
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
          />
        </div>
      )}

      {metric.type === 'summary' && (
        <div className="space-y-2">
          <Label>Objectives</Label>
          <Input
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
                if (!isNaN(k) && !isNaN(v)) objectives[k] = v
              })
              setMetric({ ...metric, objectives: Object.keys(objectives).length > 0 ? objectives : undefined })
            }}
            placeholder="0.5:0.05, 0.9:0.01, 0.99:0.001"
          />
        </div>
      )}

      <div className="flex justify-end space-x-2">
        <Button type="button" variant="outline" onClick={onCancel}>
          取消
        </Button>
        <Button
          type="submit"
          disabled={saveMutation.isPending}
        >
          {saveMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          {saveMutation.isPending ? '保存中…' : '保存'}
        </Button>
      </div>
      <AlertDialog open={showSaveConfirm} onOpenChange={setShowSaveConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认保存指标？</AlertDialogTitle>
            <AlertDialogDescription>
              <div className="mt-4 space-y-2 text-sm text-foreground bg-muted p-4 rounded-md">
                <div className="grid grid-cols-[80px_1fr] gap-2">
                  <span className="text-muted-foreground">名称:</span>
                  <span className="font-medium">{metric.name}</span>

                  <span className="text-muted-foreground">类型:</span>
                  <span className="font-medium">{metric.type}</span>

                  <span className="text-muted-foreground">数据源:</span>
                  <span className="font-medium">
                    {metric.source}
                    {metric.connection && <span className="text-muted-foreground ml-1">({metric.connection})</span>}
                  </span>

                  <span className="text-muted-foreground">Help:</span>
                  <span className="font-medium">{metric.help}</span>

                  <span className="text-muted-foreground">查询:</span>
                  <code className="font-mono text-xs break-all bg-background p-1 rounded border">
                    {metric.query}
                  </code>
                </div>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmSave}>确认保存</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </form>
  )
}
