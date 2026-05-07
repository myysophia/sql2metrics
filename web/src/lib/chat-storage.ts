const STORAGE_KEY = 'ai-chat-history'
const MAX_MESSAGES = 100
const EXPIRY_DAYS = 7

export interface Message {
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}

export interface ChatHistory {
  threadId: string | null
  messages: Message[]
  timestamp: number
}

export function saveMessagesToLocalStorage(threadId: string | null, messages: Message[]) {
  try {
    // 限制消息数量
    const trimmedMessages = messages.slice(-MAX_MESSAGES)

    const history: ChatHistory = {
      threadId,
      messages: trimmedMessages,
      timestamp: Date.now()
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(history))
  } catch (error) {
    console.error('保存聊天历史失败:', error)
  }
}

export function loadMessagesFromLocalStorage(): ChatHistory {
  try {
    const data = localStorage.getItem(STORAGE_KEY)
    if (!data) {
      return { threadId: null, messages: [], timestamp: 0 }
    }

    const history = JSON.parse(data) as ChatHistory

    // 检查过期
    const EXPIRY_MS = EXPIRY_DAYS * 24 * 60 * 60 * 1000
    if (Date.now() - history.timestamp > EXPIRY_MS) {
      clearChatHistory()
      return { threadId: null, messages: [], timestamp: 0 }
    }

    // 转换时间戳
    history.messages = history.messages.map(msg => ({
      ...msg,
      timestamp: new Date(msg.timestamp)
    }))

    return history
  } catch (error) {
    console.error('加载聊天历史失败:', error)
    return { threadId: null, messages: [], timestamp: 0 }
  }
}

export function clearChatHistory() {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch (error) {
    console.error('清除聊天历史失败:', error)
  }
}
