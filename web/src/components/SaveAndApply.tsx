import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import { api } from '../api/client'

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
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold mb-1">保存并应用配置</h3>
          <p className="text-sm text-gray-600">
            验证配置后，将触发热更新并自动打开 metrics 端点
          </p>
        </div>
        <button
          onClick={() => saveAndApplyMutation.mutate()}
          disabled={saveAndApplyMutation.isPending}
          className="px-6 py-3 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed focus-visible-ring font-medium flex items-center space-x-2"
        >
          {saveAndApplyMutation.isPending ? (
            <>
              <svg className="animate-spin h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              <span>应用中…</span>
            </>
          ) : (
            <span>保存并应用</span>
          )}
        </button>
      </div>

      <AnimatePresence>
        {saveAndApplyMutation.isError && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="mt-4 p-4 bg-red-50 border border-red-200 rounded-lg text-red-800"
          >
            <div className="font-medium">应用失败</div>
            <div className="text-sm mt-1">
              {saveAndApplyMutation.error instanceof Error
                ? saveAndApplyMutation.error.message
                : '未知错误'}
            </div>
          </motion.div>
        )}

        {showSuccess && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="mt-4 p-4 bg-green-50 border border-green-200 rounded-lg text-green-800"
          >
            <div className="flex items-center space-x-2">
              <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                  clipRule="evenodd"
                />
              </svg>
              <div>
                <div className="font-medium">配置已应用成功！</div>
                <div className="text-sm mt-1">
                  {saveAndApplyMutation.data?.reload?.message || 'Metrics 端点已在新标签页打开'}
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
