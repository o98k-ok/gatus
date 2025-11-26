#!/bin/bash
set -e

echo "🚀 Gatus 完整构建脚本"
echo "===================="

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: 构建前端
echo -e "${BLUE}📦 Step 1: 构建前端...${NC}"
cd web/app

if [ ! -d "node_modules" ]; then
    echo "  📥 安装 npm 依赖..."
    npm install
fi

echo "  🔨 构建 Vue.js 项目..."
npm run build

cd ../..

echo -e "${GREEN}✅ 前端构建完成 → web/static/${NC}"

# 验证前端构建产物
if [ ! -f "web/static/index.html" ]; then
    echo -e "${RED}❌ 错误：web/static/index.html 不存在！${NC}"
    exit 1
fi

# Step 2: 构建后端
echo -e "${BLUE}🔧 Step 2: 构建后端（包含前端 embed）...${NC}"
go mod tidy
if [ "$1" == "linux" ]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o gatus .
else 
    go build -a -installsuffix cgo -o gatus .
fi

echo -e "${GREEN}✅ 后端构建完成 → ./gatus${NC}"

# 显示二进制大小
SIZE=$(du -h gatus | cut -f1)
echo -e "${GREEN}📊 二进制大小: ${SIZE}${NC}"

# Step 3: 验证
echo -e "${BLUE}🔍 Step 3: 验证构建...${NC}"

if [ ! -f "gatus" ]; then
    echo -e "${RED}❌ 错误：gatus 二进制不存在！${NC}"
    exit 1
fi

if [ ! -x "gatus" ]; then
    chmod +x gatus
    echo "  ✅ 添加执行权限"
fi

echo ""
echo -e "${GREEN}🎉 构建成功！${NC}"
echo ""
echo "运行命令："
echo "  ./gatus -config config.yaml"
echo ""
echo "测试 Metric 功能："
echo "  ./gatus -config config-test-metric.yaml"
echo ""
echo "Docker 构建（使用修复后的 Dockerfile）："
echo "  docker build -f Dockerfile.fixed -t gatus:latest ."
echo ""

