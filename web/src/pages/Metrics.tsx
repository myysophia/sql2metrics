import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { MetricSpec } from '../types/config'
import type { TimeRangePreset, RefreshInterval } from '../types/timeseries'
import MetricForm from '../components/MetricForm'
import SaveAndApply from '../components/SaveAndApply'
import MetricChart from '../components/MetricChart'
import MetricSelector from '../components/MetricSelector'
import TimeRangeSelector from '../components/TimeRangeSelector'
import RefreshControl from '../components/RefreshControl'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Plus, Edit2, Trash2, Download } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'
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

export default function Metrics() {
  const queryClient = useQueryClient()
  const { toast } = useToast()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  // 图表相关状态
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([])
  const [timeRange, setTimeRange] = useState<TimeRangePreset>('1h')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(30)

  // 获取可用指标列表（从配置中获取）
  const availableMetrics = config?.metrics.map(m => m.name) || []

  // 查询时序数据
  const { data, refetch, isFetching, error } = useQuery({
    queryKey: ['timeseries', selectedMetrics, timeRange],
    queryFn: () =>
      api.queryTimeseries({
        metrics: selectedMetrics,
        start: `-${timeRange}`, // 添加前缀 "-" 以符合 Prometheus 格式
        end: 'now',
      }),
    enabled: selectedMetrics.length > 0,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
  })

  const [editingMetric, setEditingMetric] = useState<MetricSpec | null>(null)
  const [editingIndex, setEditingIndex] = useState<number | undefined>(undefined)
  const [isCreating, setIsCreating] = useState(false)

  const deleteMutation = useMutation({
    mutationFn: (index: number) => api.deleteMetricByIndex(index),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
    },
  })

  const handleExportCSV = async () => {
    if (selectedMetrics.length === 0) {
      toast({
        title: '导出失败',
        description: '请先选择要导出的指标',
        variant: 'destructive',
      })
      return
    }

    try {
      const blob = await api.exportTimeseries({
        metrics: selectedMetrics,
        start: `-${timeRange}`,
        end: 'now',
      })

      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `metrics-${Date.now()}.csv`
      a.click()
      URL.revokeObjectURL(url)

      toast({
        title: '导出成功',
        description: 'CSV 文件已下载',
      })
    } catch (error) {
      toast({
        title: '导出失败',
        description: error instanceof Error ? error.message : '未知错误',
        variant: 'destructive',
      })
    }
  }

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }



  return (
    <div className="space-y-8">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold tracking-tight">指标管理</h2>
        <Button
          onClick={() => {
            setIsCreating(true)
            setEditingIndex(undefined)
            setEditingMetric({
              name: '',
              help: '',
              type: 'gauge',
              source: 'mysql',
              query: '',
            })
            window.scrollTo({ top: 0, behavior: 'smooth' })
          }}
          disabled={isCreating || !!editingMetric}
        >
          <Plus className="mr-2 h-4 w-4" /> 添加指标
        </Button>
      </div>

      {(editingMetric || isCreating) && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>{isCreating ? '创建新指标' : '编辑指标'}</CardTitle>
          </CardHeader>
          <CardContent>
            <MetricForm
              metric={editingMetric!}
              metricIndex={editingIndex}
              config={config}
              onSave={() => {
                setEditingMetric(null)
                setEditingIndex(undefined)
                setIsCreating(false)
                queryClient.invalidateQueries({ queryKey: ['config'] })
                window.scrollTo({ top: 0, behavior: 'smooth' })
              }}
              onCancel={() => {
                setEditingMetric(null)
                setEditingIndex(undefined)
                setIsCreating(false)
              }}
            />
          </CardContent>
        </Card>
      )}

      <div className="rounded-md border bg-white">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>数据源</TableHead>
              <TableHead>帮助信息</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {config.metrics.map((metric, index) => (
              <TableRow key={`${metric.name}-${index}`}>
                <TableCell className="font-medium">{metric.name}</TableCell>
                <TableCell>{metric.type}</TableCell>
                <TableCell>
                  {metric.source}
                  {metric.connection && ` (${metric.connection})`}
                </TableCell>
                <TableCell>{metric.help}</TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => {
                        setEditingMetric(metric)
                        setEditingIndex(index)
                        setIsCreating(false)
                        window.scrollTo({ top: 0, behavior: 'smooth' })
                      }}
                      disabled={isCreating || !!editingMetric}
                    >
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-destructive hover:text-destructive"
                          disabled={isCreating || !!editingMetric}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>
                            确定要删除指标 "{metric.name}" 吗？
                            <span className="block text-sm font-normal text-muted-foreground mt-1">
                              数据源: {metric.source} {metric.connection ? `(${metric.connection})` : ''}
                            </span>
                          </AlertDialogTitle>
                          <AlertDialogDescription>
                            此操作不可撤销。这将从配置中永久删除该指标，并停止相关的数据采集。
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>取消</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={() => deleteMutation.mutate(index)}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                          >
                            删除
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      {/* 图表区域 */}
      <div className="space-y-6 mt-8">
        <div className="border-t pt-6">
          <h3 className="text-xl font-semibold mb-4">指标图表</h3>

          {/* 控制栏 */}
          <div className="rounded-lg border bg-card p-4 mb-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {/* 指标选择 */}
              <div className="space-y-2">
                <label className="text-sm font-medium">选择指标</label>
                <MetricSelector
                  availableMetrics={availableMetrics}
                  selectedMetrics={selectedMetrics}
                  onChange={setSelectedMetrics}
                />
              </div>

              {/* 时间范围 */}
              <div className="space-y-2">
                <label className="text-sm font-medium">时间范围</label>
                <TimeRangeSelector value={timeRange} onChange={setTimeRange} />
              </div>

              {/* 刷新控制 */}
              <div className="space-y-2">
                <label className="text-sm font-medium">刷新控制</label>
                <RefreshControl
                  autoRefresh={autoRefresh}
                  onAutoRefreshChange={setAutoRefresh}
                  refreshInterval={refreshInterval}
                  onRefreshIntervalChange={setRefreshInterval}
                  onManualRefresh={() => refetch()}
                  isRefreshing={isFetching}
                />
              </div>
            </div>
          </div>

          {/* 图表显示区域 */}
          <div className="rounded-lg border bg-card p-6">
            <MetricChart
              data={data?.data || []}
              loading={isFetching && !data}
              error={error as Error | null}
            />
          </div>

          {/* 导出栏 */}
          <div className="flex justify-end gap-2 mt-4">
            <Button
              variant="outline"
              onClick={handleExportCSV}
              disabled={selectedMetrics.length === 0}
            >
              <Download className="mr-2 h-4 w-4" />
              导出 CSV
            </Button>
          </div>

          {/* 提示信息 */}
          {selectedMetrics.length === 0 && (
            <div className="rounded-lg border bg-muted/50 p-8 text-center">
              <p className="text-muted-foreground">
                请从上方选择一个或多个指标，然后选择时间范围来查看图表
              </p>
            </div>
          )}
        </div>
      </div>

      <div className="mt-6">
        <SaveAndApply />
      </div>
    </div>
  )
}
