import { useRef } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { format } from 'date-fns'
import type { TimeseriesData } from '../types/timeseries'

interface MetricChartProps {
  data: TimeseriesData[]
  loading?: boolean
  error?: Error | null
  onExport?: () => void
}

export default function MetricChart({ data, loading, error, onExport }: MetricChartProps) {
  const chartRef = useRef<HTMLDivElement>(null)

  if (loading) {
    return (
      <div className="flex items-center justify-center h-[400px] border rounded-lg bg-muted/20">
        <div className="text-muted-foreground">加载中...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-[400px] border rounded-lg bg-destructive/10">
        <div className="text-destructive font-medium mb-2">查询失败</div>
        <div className="text-sm text-muted-foreground">{error.message}</div>
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[400px] border rounded-lg bg-muted/20">
        <div className="text-center">
          <div className="text-muted-foreground mb-2">请选择指标和时间范围查看图表</div>
          <div className="text-xs text-muted-foreground/60">
            提示：确保采集服务正在运行，并且所选时间范围内有数据
          </div>
        </div>
      </div>
    )
  }

  // 转换数据为 Recharts 格式
  const chartData: Record<string, string | number | null>[] = []
  const timestamps = new Set<number>()

  // 收集所有时间戳
  data.forEach((series) => {
    series.values.forEach(([ts]) => timestamps.add(ts))
  })

  // 排序时间戳
  const sortedTimestamps = Array.from(timestamps).sort((a, b) => a - b)

  // 构建图表数据
  sortedTimestamps.forEach((ts) => {
    const point: Record<string, string | number | null> = {
      timestamp: format(new Date(ts * 1000), 'HH:mm:ss'),
    }

    data.forEach((series) => {
      const metricName = series.metric.__name__ || Object.values(series.metric)[0] || 'unknown'
      const value = series.values.find(([timestamp]) => timestamp === ts)?.[1]
      point[metricName] = value ?? null
    })

    chartData.push(point)
  })

  // 生成颜色
  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899']

  return (
    <div ref={chartRef} className="space-y-4">
      <div className="flex justify-between items-center">
        <div className="text-sm text-muted-foreground">
          显示 {data.length} 个指标，{chartData.length} 个数据点
        </div>
        {onExport && (
          <button
            onClick={onExport}
            className="px-3 py-1 text-sm border rounded hover:bg-muted"
          >
            导出图片
          </button>
        )}
      </div>

      <ResponsiveContainer width="100%" height={400}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis
            dataKey="timestamp"
            className="text-xs"
            stroke="hsl(var(--muted-foreground))"
          />
          <YAxis
            className="text-xs"
            stroke="hsl(var(--muted-foreground))"
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--background))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '6px',
            }}
            labelStyle={{ color: 'hsl(var(--foreground))' }}
          />
          <Legend />
          {data.map((series, index) => {
            const metricName = series.metric.__name__ || Object.values(series.metric)[0] || 'unknown'
            return (
              <Line
                key={metricName}
                type="monotone"
                dataKey={metricName}
                stroke={colors[index % colors.length]}
                strokeWidth={2}
                dot={false}
                connectNulls={false}
              />
            )
          })}
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
