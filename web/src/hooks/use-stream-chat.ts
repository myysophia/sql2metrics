import { useRef, useCallback, useState } from 'react'

interface StreamChatOptions {
  onChunk?: (content: string) => void
  onToolCall?: (tool: string, args: any) => void
  onComplete?: (threadId: string, fullMessage: string) => void
  onError?: (error: Error) => void
}

interface StreamChatResult {
  sendMessage: (message: string, threadId: string | null) => Promise<void>
  isLoading: boolean
  abort: () => void
}

export function useStreamChat(options: StreamChatOptions = {}): StreamChatResult {
  const abortControllerRef = useRef<AbortController | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const sendMessage = useCallback(async (message: string, threadId: string | null) => {
    // 取消之前的请求
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    abortControllerRef.current = new AbortController()
    setIsLoading(true)

    console.log('🔵 开始发送请求:', { message, threadId })

    try {
      const response = await fetch('/api/ai/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message,
          thread_id: threadId
        }),
        signal: abortControllerRef.current.signal
      })

      console.log('🟢 响应状态:', response.status, response.statusText)

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      const decoder = new TextDecoder()

      if (!reader) {
        throw new Error('无法读取响应流')
      }

      let buffer = ''
      let fullContent = ''
      let chunkCount = 0

      while (true) {
        const { done, value } = await reader.read()

        if (done) {
          console.log('⏹️ 流式读取完成, 共', chunkCount, '个 chunks')
          break
        }

        // 解码数据块
        buffer += decoder.decode(value, { stream: true })
        chunkCount++

        // 处理 SSE 格式：event: xxx\ndata: xxx\n\n
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''  // 保留未完成的行

        let eventType = ''
        let eventData = ''

        for (let i = 0; i < lines.length; i++) {
          const line = lines[i].trim()

          if (!line) {
            // 空行表示一个事件块结束，处理累积的 event 和 data
            if (eventType && eventData) {
              try {
                const data = JSON.parse(eventData)

                if (eventType === 'chunk') {
                  fullContent += data.content
                  options.onChunk?.(data.content)
                } else if (eventType === 'tool_call') {
                  console.log('🔧 工具调用:', data)
                  options.onToolCall?.(data.tool, data.args)
                } else if (eventType === 'done') {
                  console.log('✅ 完成事件:', data)
                  options.onComplete?.(data.thread_id, fullContent)
                } else if (eventType === 'error') {
                  console.error('❌ 错误事件:', data)
                  options.onError?.(new Error(data.error))
                }
              } catch (e) {
                console.error('解析 SSE 数据失败:', e, 'Raw data:', eventData)
              }
            }
            // 重置
            eventType = ''
            eventData = ''
          } else if (line.startsWith('event: ')) {
            eventType = line.substring(7).trim()
          } else if (line.startsWith('data: ')) {
            eventData = line.substring(6).trim()
          }
        }
      }
    } catch (error) {
      console.error('❌ 请求失败:', error)
      if (error instanceof Error && error.name === 'AbortError') {
        console.log('请求已取消')
      } else {
        options.onError?.(error as Error)
      }
    } finally {
      setIsLoading(false)
      abortControllerRef.current = null
    }
  }, [options])

  const abort = useCallback(() => {
    abortControllerRef.current?.abort()
    setIsLoading(false)
  }, [])

  return {
    sendMessage,
    isLoading,
    abort
  }
}
