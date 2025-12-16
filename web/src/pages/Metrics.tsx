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

export default function Metrics() {
  const queryClient = useQueryClient()
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  const [editingMetric, setEditingMetric] = useState<MetricSpec | null>(null)
  const [isCreating, setIsCreating] = useState(false)

  const deleteMutation = useMutation({
    mutationFn: (name: string) => api.deleteMetric(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['config'] })
    },
  })

  if (isLoading || !config) {
    return <div className="text-center py-12">加载中…</div>
  }

  const handleDelete = (name: string) => {
    if (!confirm(`确定要删除指标 "${name}" 吗？`)) return
    deleteMutation.mutate(name)
  }

  return (
    <div className="space-y-8">
      <div className="flex justify-between items-center">
        <h2 className="text-3xl font-bold tracking-tight">指标管理</h2>
        <Button
          onClick={() => {
            setIsCreating(true)
            setEditingMetric({
              name: '',
              help: '',
              type: 'gauge',
              source: 'mysql',
              query: '',
            })
          }}
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
              config={config}
              onSave={() => {
                setEditingMetric(null)
                setIsCreating(false)
                queryClient.invalidateQueries({ queryKey: ['config'] })
              }}
              onCancel={() => {
                setEditingMetric(null)
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
            {config.metrics.map((metric) => (
              <TableRow key={metric.name}>
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
                        setIsCreating(false)
                      }}
                    >
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleDelete(metric.name)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
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
