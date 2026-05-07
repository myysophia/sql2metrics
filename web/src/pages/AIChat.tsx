import { useState, useEffect, useRef } from 'react'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Sparkles, Send, Loader2, Trash2 } from 'lucide-react'
import { useStreamChat } from '@/hooks/use-stream-chat'
import { saveMessagesToLocalStorage, loadMessagesFromLocalStorage, clearChatHistory } from '@/lib/chat-storage'

interface Message {
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
  isStreaming?: boolean
}

export default function AIChat() {
  const [messages, setMessages] = useState<Message[]>([])
  const [input, setInput] = useState('')
  const [threadId, setThreadId] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

  // 集成流式聊天 Hook
  const { sendMessage, isLoading } = useStreamChat({
    onChunk: (content) => {
      console.log('📥 收到 chunk:', content)
      setMessages(prev => {
        const last = prev[prev.length - 1]
        if (last?.role === 'assistant' && last?.isStreaming) {
          // 追加内容到正在流式输出的消息
          return [...prev.slice(0, -1), { ...last, content: last.content + content }]
        }
        // 创建新的流式消息
        return [...prev, {
          role: 'assistant',
          content,
          timestamp: new Date(),
          isStreaming: true
        }]
      })
    },
    onToolCall: (tool, args) => {
      console.log('🔧 工具调用:', tool, args)
      // 显示工具调用消息
      const toolMessage = `🔧 正在执行: ${tool}\n\`\`\`\n${JSON.stringify(args, null, 2)}\n\`\`\``
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: toolMessage,
        timestamp: new Date(),
        isStreaming: false
      }])
    },
    onComplete: (newThreadId, fullMessage) => {
      console.log('✅ 流式完成:', { newThreadId, fullMessage })
      setMessages(prev => {
        const last = prev[prev.length - 1]
        if (last?.role === 'assistant' && last?.isStreaming) {
          // 标记流式输出完成
          const updated = [...prev.slice(0, -1), { ...last, isStreaming: false }]
          // 保存到 localStorage（使用更新后的消息列表）
          saveMessagesToLocalStorage(newThreadId, updated)
          return updated
        }
        // 如果没有流式消息，直接保存
        saveMessagesToLocalStorage(newThreadId, prev)
        return prev
      })
      setThreadId(newThreadId)
    },
    onError: (error) => {
      console.error('❌ 流式错误:', error)
      setMessages(prev => [...prev, {
        role: 'assistant',
        content: `错误: ${error.message}`,
        timestamp: new Date()
      }])
    }
  })

  // 页面加载时恢复历史消息
  useEffect(() => {
    const saved = loadMessagesFromLocalStorage()
    if (saved.messages.length > 0) {
      setMessages(saved.messages)
      setThreadId(saved.threadId)
    }
  }, [])

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages, isLoading])

  const handleSend = async () => {
    if (!input.trim() || isLoading) return

    console.log('🚀 准备发送消息:', input.trim())
    const userMessage: Message = {
      role: 'user',
      content: input.trim(),
      timestamp: new Date()
    }

    setMessages(prev => [...prev, userMessage])
    const messageToSend = input.trim()
    setInput('')

    console.log('📤 调用 sendMessage, threadId:', threadId)
    // 使用流式发送
    await sendMessage(messageToSend, threadId)
  }

  const handleClearHistory = () => {
    clearChatHistory()
    setMessages([])
    setThreadId(null)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col">
      {/* Header */}
      <div className="mb-6 flex-shrink-0 flex items-start justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Sparkles className="h-8 w-8 text-purple-600" />
            AI 智能助手
          </h1>
          <p className="text-muted-foreground mt-2">
            通过自然语言查询指标、创建配置、分析数据
          </p>
        </div>
        {messages.length > 0 && (
          <Button
            variant="outline"
            size="sm"
            onClick={handleClearHistory}
            className="gap-2"
          >
            <Trash2 className="h-4 w-4" />
            清除历史
          </Button>
        )}
      </div>

      {/* Chat Card */}
      <Card className="flex-1 flex flex-col overflow-hidden">
        {/* Messages Area */}
        <ScrollArea className="flex-1 p-4">
          {messages.length === 0 ? (
            <div className="flex items-center justify-center h-full text-muted-foreground">
              <div className="text-center">
                <Sparkles className="h-12 w-12 mx-auto mb-4 text-purple-400" />
                <p className="mb-2 font-medium">开始与 AI 助手对话</p>
                <p className="text-sm text-muted-foreground">
                  试试问："帮我列出所有指标" 或 "创建一个新指标"
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {messages.map((msg, idx) => (
                <div
                  key={idx}
                  className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[80%] rounded-lg px-4 py-3 ${
                      msg.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted'
                    }`}
                  >
                    {/* Simple markdown-like formatting for code blocks */}
                    {msg.content.includes('```') ? (
                      <div className="space-y-2">
                        {msg.content.split('```').map((part, i) => {
                          if (i % 2 === 1) {
                            // Code block - first line is language, rest is code
                            const lines = part.split('\n')
                            const code = lines.slice(1).join('\n')
                            return (
                              <pre key={i} className="bg-black/20 p-3 rounded overflow-x-auto text-sm">
                                <code>{code}</code>
                              </pre>
                            )
                          }
                          return <div key={i} className="whitespace-pre-wrap">{part}</div>
                        })}
                      </div>
                    ) : (
                      <div className="whitespace-pre-wrap">{msg.content}</div>
                    )}
                    <div className="text-xs opacity-70 mt-2">
                      {msg.timestamp.toLocaleTimeString()}
                    </div>
                  </div>
                </div>
              ))}
              {isLoading && (
                <div className="flex justify-start">
                  <div className="bg-muted rounded-lg px-4 py-3 flex items-center gap-2">
                    <Loader2 className="h-5 w-5 animate-spin" />
                    <span className="text-sm text-muted-foreground">AI 正在思考...</span>
                  </div>
                </div>
              )}
              <div ref={scrollRef} />
            </div>
          )}
        </ScrollArea>

        {/* Input Area */}
        <div className="border-t p-4">
          <div className="flex gap-2">
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入你的问题... (Enter 发送，Shift+Enter 换行)"
              className="min-h-[60px] resize-none"
              disabled={isLoading}
              autoFocus
            />
            <Button
              onClick={handleSend}
              disabled={!input.trim() || isLoading}
              size="icon"
              className="h-[60px] w-[60px] flex-shrink-0"
            >
              {isLoading ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <Send className="h-5 w-5" />
              )}
            </Button>
          </div>
          <div className="text-xs text-muted-foreground mt-2 flex justify-between">
            <span>{input.length} 字符</span>
            <span>Shift+Enter 换行</span>
          </div>
        </div>
      </Card>
    </div>
  )
}
