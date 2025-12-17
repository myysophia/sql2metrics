import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api } from '../api/client'
import type { MetricSpec } from '../types/config'
import MetricForm from '../components/MetricForm'
import SaveAndApply from '../components/SaveAndApply'
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
import { Plus, Edit2, Trash2 } from 'lucide-react'
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
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
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

      <div className="mt-6">
        <SaveAndApply />
      </div>
    </div>
  )
}
