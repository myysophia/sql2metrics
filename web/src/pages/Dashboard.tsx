import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Activity, Database, Server, Clock, ExternalLink, Zap, BarChart3, Settings2, Shield, Workflow } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { motion } from 'framer-motion'

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1
    }
  }
}

const itemVariants = {
  hidden: { y: 20, opacity: 0 },
  visible: {
    y: 0,
    opacity: 1,
    transition: {
      type: 'spring',
      stiffness: 100
    }
  }
}

const featureCardVariants = {
  hidden: { scale: 0.9, opacity: 0 },
  visible: {
    scale: 1,
    opacity: 1,
    transition: {
      type: 'spring',
      stiffness: 100
    }
  },
  hover: {
    scale: 1.02,
    boxShadow: '0 10px 30px rgba(0,0,0,0.1)',
    transition: {
      type: 'spring',
      stiffness: 400,
      damping: 10
    }
  }
}

const iconPulseVariants = {
  hover: {
    scale: [1, 1.2, 1],
    transition: {
      duration: 0.6,
      repeat: Infinity,
      repeatType: 'loop' as const
    }
  }
}

export default function Dashboard() {
  const { data: config, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: api.getConfig,
  })

  if (isLoading) {
    return <div className="text-center py-12">加载中…</div>
  }

  if (!config) {
    return <div className="text-center py-12 text-red-600">加载配置失败</div>
  }

  const listenHost = config.prometheus.listen_address === '0.0.0.0' ? 'localhost' : config.prometheus.listen_address
  const metricsURL = `http://${listenHost}:${config.prometheus.listen_port}/metrics`

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">概览</h2>

      <motion.div
        className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8"
        variants={containerVariants}
        initial="hidden"
        animate="visible"
      >
        <motion.div variants={itemVariants}>
          <Card className="h-full">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">指标总数</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <motion.div
                className="text-2xl font-bold tabular-nums"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 200, delay: 0.2 }}
              >
                {config.metrics.length}
              </motion.div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div variants={itemVariants}>
          <Card className="h-full">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">MySQL 连接</CardTitle>
              <Database className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <motion.div
                className="text-2xl font-bold tabular-nums"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 200, delay: 0.3 }}
              >
                {Object.keys(config.mysql_connections || {}).length}
              </motion.div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div variants={itemVariants}>
          <Card className="h-full">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Redis 连接</CardTitle>
              <Server className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <motion.div
                className="text-2xl font-bold tabular-nums"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 200, delay: 0.4 }}
              >
                {Object.keys(config.redis_connections || {}).length}
              </motion.div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div variants={itemVariants}>
          <Card className="h-full">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">采集周期</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <motion.div
                className="text-2xl font-bold"
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ type: 'spring', stiffness: 200, delay: 0.5 }}
              >
                {config.schedule.interval}
              </motion.div>
            </CardContent>
          </Card>
        </motion.div>
      </motion.div>

      <Card>
        <CardHeader>
          <CardTitle>Metrics 端点</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-4">
            <code className="flex-1 bg-muted px-4 py-2 rounded font-mono text-sm">{metricsURL}</code>
            <Button asChild>
              <a
                href={metricsURL}
                target="_blank"
                rel="noopener noreferrer"
              >
                <ExternalLink className="mr-2 h-4 w-4" />
                打开
              </a>
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 项目亮点 */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.6, duration: 0.5 }}
      >
        <Card className="mt-8">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <motion.span
                animate={{ rotate: [0, 15, -15, 0] }}
                transition={{ duration: 2, repeat: Infinity, repeatDelay: 3 }}
              >
                🌟
              </motion.span>
              项目介绍
            </CardTitle>
            <CardDescription>SQL2Metrics 是一款强大的数据库指标采集工具，将 SQL 查询结果转换为 Prometheus 指标</CardDescription>
          </CardHeader>
          <CardContent>
            <motion.div
              className="grid gap-4 md:grid-cols-2 lg:grid-cols-3"
              variants={containerVariants}
              initial="hidden"
              animate="visible"
            >
              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-primary/10" variants={iconPulseVariants}>
                  <Database className="h-5 w-5 text-primary" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">多数据源支持</h4>
                  <p className="text-sm text-muted-foreground">支持 MySQL、Redis、IoTDB 等多种数据源，轻松接入现有基础设施</p>
                </div>
              </motion.div>

              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-blue-500/10" variants={iconPulseVariants}>
                  <BarChart3 className="h-5 w-5 text-blue-500" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">灵活的指标类型</h4>
                  <p className="text-sm text-muted-foreground">支持 Gauge、Counter、Histogram、Summary 四种 Prometheus 指标类型</p>
                </div>
              </motion.div>

              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-green-500/10" variants={iconPulseVariants}>
                  <Zap className="h-5 w-5 text-green-500" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">高性能采集</h4>
                  <p className="text-sm text-muted-foreground">Go 语言编写，并发采集，支持连接池，高效处理大量指标</p>
                </div>
              </motion.div>

              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-orange-500/10" variants={iconPulseVariants}>
                  <Settings2 className="h-5 w-5 text-orange-500" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">可视化配置</h4>
                  <p className="text-sm text-muted-foreground">Web 管理界面，无需手动编辑 YAML，实时预览和热更新</p>
                </div>
              </motion.div>

              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-purple-500/10" variants={iconPulseVariants}>
                  <Workflow className="h-5 w-5 text-purple-500" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">动态标签</h4>
                  <p className="text-sm text-muted-foreground">支持从查询结果动态提取标签，构建多维度监控指标</p>
                </div>
              </motion.div>

              <motion.div
                className="flex items-start gap-3 p-4 rounded-lg border bg-card cursor-pointer"
                variants={featureCardVariants}
                whileHover="hover"
              >
                <motion.div className="p-2 rounded-md bg-red-500/10" variants={iconPulseVariants}>
                  <Shield className="h-5 w-5 text-red-500" />
                </motion.div>
                <div>
                  <h4 className="font-semibold">生产就绪</h4>
                  <p className="text-sm text-muted-foreground">Docker 部署、优雅关闭、配置校验，开箱即用</p>
                </div>
              </motion.div>
            </motion.div>
          </CardContent>
        </Card>
      </motion.div>

      {/* AI Roadmap */}
      <motion.div
        initial={{ opacity: 0, y: 30 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.8, duration: 0.5 }}
      >
        <Card className="mt-8 border-dashed border-2 border-violet-300">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <motion.span
                animate={{ scale: [1, 1.2, 1] }}
                transition={{ duration: 1.5, repeat: Infinity, repeatDelay: 2 }}
              >
                🤖
              </motion.span>
              AI Roadmap
              <span className="ml-2 px-2 py-0.5 text-xs font-normal bg-violet-100 text-violet-700 rounded-full">Coming Soon</span>
            </CardTitle>
            <CardDescription>探索 AI 与监控的结合，让 SQL2Metrics 更智能</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-3 md:grid-cols-2">
              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-violet-50 to-transparent border border-violet-200">
                <div className="p-1.5 rounded bg-violet-100 text-violet-600 text-lg">💬</div>
                <div>
                  <h4 className="font-medium text-sm">自然语言配置生成</h4>
                  <p className="text-xs text-muted-foreground">用自然语言描述监控需求，AI 自动生成 SQL 和指标配置</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-blue-50 to-transparent border border-blue-200">
                <div className="p-1.5 rounded bg-blue-100 text-blue-600 text-lg">🔍</div>
                <div>
                  <h4 className="font-medium text-sm">异常检测与根因分析</h4>
                  <p className="text-xs text-muted-foreground">AI 自动识别指标异常，分析关联指标并给出可能原因</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-green-50 to-transparent border border-green-200">
                <div className="p-1.5 rounded bg-green-100 text-green-600 text-lg">🎯</div>
                <div>
                  <h4 className="font-medium text-sm">智能告警优化</h4>
                  <p className="text-xs text-muted-foreground">动态阈值调整、告警聚合降噪、优先级智能排序</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-orange-50 to-transparent border border-orange-200">
                <div className="p-1.5 rounded bg-orange-100 text-orange-600 text-lg">🔗</div>
                <div>
                  <h4 className="font-medium text-sm">MCP Server 集成</h4>
                  <p className="text-xs text-muted-foreground">封装为 MCP 工具，支持 AI Agent 对话式查询监控数据</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-pink-50 to-transparent border border-pink-200">
                <div className="p-1.5 rounded bg-pink-100 text-pink-600 text-lg">📈</div>
                <div>
                  <h4 className="font-medium text-sm">预测性监控</h4>
                  <p className="text-xs text-muted-foreground">基于历史趋势预测资源容量和业务指标走势</p>
                </div>
              </div>

              <div className="flex items-start gap-3 p-3 rounded-lg bg-gradient-to-r from-cyan-50 to-transparent border border-cyan-200">
                <div className="p-1.5 rounded bg-cyan-100 text-cyan-600 text-lg">🧠</div>
                <div>
                  <h4 className="font-medium text-sm">知识库问答</h4>
                  <p className="text-xs text-muted-foreground">基于监控数据构建知识库，支持自然语言查询历史趋势</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}
