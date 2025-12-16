import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../api/client'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Loader2, CheckCircle2, AlertCircle } from 'lucide-react'

export default function SaveAndApply() {
  const queryClient = useQueryClient()
  const [showSuccess, setShowSuccess] = useState(false)

  const saveAndApplyMutation = useMutation({
    mutationFn: async () => {
      // 获取当前配置
      const config = await api.getConfig()
      // 验证配置
      const validation = await api.validateConfig()
      if (!validation.valid) {
        throw new Error(validation.error || '配置验证失败')
      }
      // 更新配置并触发热更新
      const result = await api.updateConfig(config)
      return result
    },
    onSuccess: async () => {
      setShowSuccess(true)
      queryClient.invalidateQueries({ queryKey: ['config'] })

      // 获取 metrics URL 并打开浏览器
      try {
        const { url } = await api.getMetricsURL()
        window.open(url, '_blank', 'noopener,noreferrer')
      } catch (error) {
        console.error('获取 metrics URL 失败:', error)
      }

      // 3秒后隐藏成功提示
      setTimeout(() => setShowSuccess(false), 3000)
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 p-6">
        <div className="space-y-1">
          <CardTitle>保存并应用配置</CardTitle>
          <CardDescription>
            验证配置后，将触发热更新并自动打开 metrics 端点
          </CardDescription>
        </div>
        <Button
          onClick={() => saveAndApplyMutation.mutate()}
          disabled={saveAndApplyMutation.isPending}
        >
          {saveAndApplyMutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              应用中…
            </>
          ) : (
            '保存并应用'
          )}
        </Button>
      </CardHeader>

      <AnimatePresence>
        {saveAndApplyMutation.isError && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="px-6 pb-6"
          >
            <div className="p-4 bg-destructive/10 text-destructive border border-destructive/20 rounded-md flex items-center gap-2">
              <AlertCircle className="h-4 w-4" />
              <div>
                <div className="font-medium">应用失败</div>
                <div className="text-sm">
                  {saveAndApplyMutation.error instanceof Error
                    ? saveAndApplyMutation.error.message
                    : '未知错误'}
                </div>
              </div>
            </div>
          </motion.div>
        )}

        {showSuccess && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="px-6 pb-6"
          >
            <div className="p-4 bg-green-50 text-green-900 border border-green-200 rounded-md flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <div>
                <div className="font-medium">配置已应用成功！</div>
                <div className="text-sm">
                  {saveAndApplyMutation.data?.reload?.message || 'Metrics 端点已在新标签页打开'}
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </Card>
  )
}
