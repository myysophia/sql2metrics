#!/bin/bash
set -e

echo "=== SQL2Metrics 构建脚本 ==="

# 1. 检查环境
echo "[1/4] 检查构建环境..."
if ! command -v go &> /dev/null; then
    echo "错误: 未找到 Go 环境"
    exit 1
fi
if ! command -v npm &> /dev/null; then
    echo "错误: 未找到 npm 环境"
    exit 1
fi

# 2. 构建前端
echo "[2/4] 构建前端资源..."
cd web
if [ ! -d "node_modules" ]; then
    echo "  安装前端依赖..."
    npm install
fi
echo "  编译前端代码..."
npm run build
cd ..

# 3. 构建后端
echo "[3/4] 编译后端服务..."
# 确保 web/dist 存在，否则 go embed 会报错
if [ ! -d "web/dist" ]; then
    echo "错误: 前端构建失败，web/dist 不存在"
    exit 1
fi

go build -o sql2metrics ./cmd/collector
if [ $? -ne 0 ]; then
    echo "错误: 后端编译失败"
    exit 1
fi

# 4. 完成
echo "[4/4] 构建完成!"
echo "运行 ./sql2metrics -config configs/config.yml 启动服务"
