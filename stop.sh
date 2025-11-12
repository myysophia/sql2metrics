#!/bin/bash

echo "=== 停止 SQL2Metrics 服务 ==="

# 停止后端
echo "停止后端服务..."
pkill -f sql2metrics
if [ $? -eq 0 ]; then
    echo "✓ 后端服务已停止"
else
    echo "⚠ 未找到后端进程"
fi

# 停止前端
echo "停止前端服务..."
pkill -f "vite\|npm.*dev"
if [ $? -eq 0 ]; then
    echo "✓ 前端服务已停止"
else
    echo "⚠ 未找到前端进程"
fi

echo "完成"
