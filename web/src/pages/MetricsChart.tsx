import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { BarChart3, Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useToast } from '@/hooks/use-toast'
import MetricChart from '../components/MetricChart'
import MetricSelector from '../components/MetricSelector'
import TimeRangeSelector from '../components/TimeRangeSelector'
import RefreshControl from '../components/RefreshControl'
import type { TimeRangePreset, RefreshInterval } from '../types/timeseries'

export default function MetricsChart() {
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>([])
  const [timeRange, setTimeRange] = useState<TimeRangePreset>('1h')
  const [autoRefresh, setAutoRefresh] = useState(false)
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(30)
  const { toast } = useToast()

  // 获取可用指标列表
  const { data: availableMetrics = [] } = useQuery({
    queryKey: ['timeseries-metrics'],
    queryFn: () => api.listAvailableMetrics(),
  })

  // 查询时序数据
  const { data, refetch, isLoading, isFetching } = useQuery({
    queryKey: ['timeseries', selectedMetrics, timeRange],
    queryFn: () =>
      api.queryTimeseries({
        metrics: selectedMetrics,
        start: timeRange,
        end: 'now',
      }),
    enabled: selectedMetrics.length > 0,
    refetchInterval: autoRefresh ? refreshInterval * 1000 : false,
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
        start: timeRange,
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

  const handleExportImage = async () => {
    toast({
      title: '功能开发中',
      description: '图片导出功能即将推出',
    })
  }

  return (
    <div className="space-y-6">
      {/* 顶部标题 */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2">
          <BarChart3 className="h-6 w-6" />
          <h2 className="text-3xl font-bold tracking-tight">指标图表</h2>
        </div>
      </div>

      {/* 控制栏 */}
      <div className="rounded-lg border bg-card p-4">
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

      {/* 图表区域 */}
      <div className="rounded-lg border bg-card p-6">
        <MetricChart
          data={data?.data || []}
          loading={isLoading}
          onExport={handleExportImage}
        />
      </div>

      {/* 导出栏 */}
      <div className="flex justify-end gap-2">
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
  )
}
