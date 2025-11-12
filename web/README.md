# SQL2Metrics Web UI

SQL2Metrics 的 Web 配置界面，提供可视化的配置管理功能。

## 功能特性

- 📊 **概览仪表板** - 查看配置概览和指标统计
- 🔌 **数据源管理** - 配置和管理 MySQL 和 IoTDB 连接
- 📈 **指标管理** - 创建、编辑和删除 Prometheus 指标
- 🔍 **SQL 预览** - 实时预览 SQL 查询结果
- ✅ **配置验证** - 实时验证配置合法性
- 🔥 **热更新** - 配置保存后自动热更新，无需重启服务
- 🚀 **一键应用** - 配置完成后自动打开浏览器展示 metrics

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build
```

## 技术栈

- React 18 + TypeScript
- Vite
- Tailwind CSS
- Monaco Editor (SQL 编辑器)
- React Query (数据获取)
- Framer Motion (动画)


