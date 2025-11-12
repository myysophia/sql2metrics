#!/bin/bash

echo "=== SQL2Metrics 启动脚本 ==="

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 环境"
    exit 1
fi

# 检查配置文件
if [ ! -f "configs/config.yml" ]; then
    echo "错误: 配置文件不存在: configs/config.yml"
    exit 1
fi

# 编译后端
echo "1. 编译后端服务..."
go build -o sql2metrics ./cmd/collector
if [ $? -ne 0 ]; then
    echo "错误: 编译失败"
    exit 1
fi

# 启动后端
echo "2. 启动后端服务..."
pkill -f sql2metrics 2>/dev/null
sleep 1

# 清除代理设置（避免数据库连接被代理拦截）
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY all_proxy ALL_PROXY 2>/dev/null

nohup env -u http_proxy -u https_proxy -u HTTP_PROXY -u HTTPS_PROXY -u all_proxy -u ALL_PROXY ./sql2metrics -config configs/config.yml > sql2metrics.log 2>&1 &
BACKEND_PID=$!
echo "   后端进程 ID: $BACKEND_PID"
echo "   已清除代理设置"

# 等待后端启动
echo "3. 等待后端启动..."
sleep 5

# 检查后端是否运行
if ps -p $BACKEND_PID > /dev/null 2>&1; then
    echo "   ✓ 后端服务运行中"
else
    echo "   ✗ 后端服务启动失败，查看日志:"
    tail -20 sql2metrics.log
    exit 1
fi

# 检查端口
if ss -tlnp 2>/dev/null | grep -q 8080 || netstat -tlnp 2>/dev/null | grep -q 8080; then
    echo "   ✓ 端口 8080 已监听"
else
    echo "   ⚠ 端口 8080 未监听（可能数据库连接失败）"
fi

# 测试 API
echo "4. 测试 API..."
if curl -s --max-time 2 http://localhost:8080/api/config > /dev/null 2>&1; then
    echo "   ✓ API 可访问"
else
    echo "   ⚠ API 暂时不可访问（可能数据库连接失败）"
fi

# 启动前端（可选）
if [ "$1" == "--with-frontend" ]; then
    echo "5. 启动前端服务..."
    cd web
    if [ ! -d "node_modules" ]; then
        echo "   安装前端依赖..."
        npm install
    fi
    npm run dev > ../frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo "   前端进程 ID: $FRONTEND_PID"
    cd ..
    sleep 3
    echo "   ✓ 前端服务启动中"
fi

echo ""
echo "=== 启动完成 ==="
echo "后端 API: http://localhost:8080/api/config"
echo "Prometheus: http://localhost:8080/metrics"
if [ "$1" == "--with-frontend" ]; then
    echo "前端界面: http://localhost:3000"
fi
echo ""
echo "查看后端日志: tail -f sql2metrics.log"
if [ "$1" == "--with-frontend" ]; then
    echo "查看前端日志: tail -f frontend.log"
fi
echo ""
echo "停止服务: pkill -f sql2metrics; pkill -f vite"
