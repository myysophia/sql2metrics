import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import { Plus, Trash2, Power, PowerOff } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { toast } from '@/hooks/use-toast'
import { useNavigate } from 'react-router-dom'

export default function Alerts() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [deleteDialog, setDeleteDialog] = useState<{ open: boolean; id: string; name: string }>({
    open: false,
    id: '',
    name: '',
  })

  const { data: alerts, isLoading } = useQuery({
    queryKey: ['alerts'],
    queryFn: () => api.listAlerts(),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
      toast({ title: '告警规则已删除' })
      setDeleteDialog({ open: false, id: '', name: '' })
    },
    onError: (error: Error) => {
      toast({ title: '删除失败', description: error.message, variant: 'destructive' })
    },
  })

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      enabled ? api.enableAlert(id) : api.disableAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: Error) => {
      toast({ title: '操作失败', description: error.message, variant: 'destructive' })
    },
  })

  const handleDelete = (id: string, name: string) => {
    setDeleteDialog({ open: true, id, name })
  }

  const handleConfirmDelete = () => {
    deleteMutation.mutate(deleteDialog.id)
  }

  const handleToggle = (id: string, enabled: boolean) => {
    toggleMutation.mutate({ id, enabled: !enabled })
  }

  const getStateBadgeClass = (state: string) => {
    switch (state) {
      case 'firing':
        return 'bg-red-100 text-red-800'
      case 'resolved':
        return 'bg-green-100 text-green-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getSeverityBadgeClass = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-100 text-red-800'
      case 'warning':
        return 'bg-yellow-100 text-yellow-800'
      case 'info':
        return 'bg-blue-100 text-blue-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  if (isLoading) {
    return <div className="text-center py-12">加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold text-foreground">告警规则</h1>
          <p className="text-muted-foreground mt-1">管理指标告警规则</p>
        </div>
        <Button onClick={() => navigate('/alerts/new')}>
          <Plus className="mr-2 h-4 w-4" />
          新建告警规则
        </Button>
      </div>

      {!alerts || alerts.length === 0 ? (
        <div className="text-center py-12 border-2 border-dashed rounded-lg">
          <p className="text-muted-foreground mb-4">还没有创建任何告警规则</p>
          <Button onClick={() => navigate('/alerts/new')}>
            <Plus className="mr-2 h-4 w-4" />
            创建第一个告警规则
          </Button>
        </div>
      ) : (
        <div className="border rounded-lg bg-background">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>指标</TableHead>
                <TableHead>类型</TableHead>
                <TableHead>严重级别</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>评估模式</TableHead>
                <TableHead>触发次数</TableHead>
                <TableHead>最后触发</TableHead>
                <TableHead>启用</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {alerts.map((alert) => (
                <TableRow
                  key={alert.id}
                  className={`cursor-pointer ${!alert.enabled ? 'opacity-50' : ''}`}
                  onClick={() => navigate(`/alerts/${alert.id}`)}
                >
                  <TableCell className="font-medium">{alert.name}</TableCell>
                  <TableCell className="font-mono text-sm">{alert.metric_name}</TableCell>
                  <TableCell className="capitalize">{alert.condition.type}</TableCell>
                  <TableCell>
                    <span
                      className={`px-2 py-1 rounded-full text-xs font-medium ${getSeverityBadgeClass(
                        alert.severity
                      )}`}
                    >
                      {alert.severity}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span
                      className={`px-2 py-1 rounded-full text-xs font-medium ${getStateBadgeClass(
                        alert.state
                      )}`}
                    >
                      {alert.state === 'pending' ? '待命' : alert.state === 'firing' ? '触发中' : '已恢复'}
                    </span>
                  </TableCell>
                  <TableCell>
                    {alert.evaluation_mode === 'collection' ? '采集时' : '定时'}
                  </TableCell>
                  <TableCell>{alert.trigger_count}</TableCell>
                  <TableCell>
                    {alert.last_triggered
                      ? new Date(alert.last_triggered).toLocaleString('zh-CN')
                      : '-'}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleToggle(alert.id, alert.enabled)
                      }}
                    >
                      {alert.enabled ? (
                        <Power className="h-4 w-4 text-green-600" />
                      ) : (
                        <PowerOff className="h-4 w-4 text-gray-400" />
                      )}
                    </Button>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleDelete(alert.id, alert.name)
                      }}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <AlertDialog open={deleteDialog.open} onOpenChange={(open: boolean) => setDeleteDialog({ ...deleteDialog, open })}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除告警规则 "{deleteDialog.name}" 吗？此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
