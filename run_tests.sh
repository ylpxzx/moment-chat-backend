#!/bin/bash

# 阅后即焚聊天 - 测试运行脚本

set -e

echo "========================================="
echo "阅后即焚聊天 - 测试套件"
echo "========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查依赖
echo -e "${YELLOW}检查依赖...${NC}"

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: Go未安装${NC}"
    exit 1
fi

# 检查Redis是否运行（可选）
if command -v redis-cli &> /dev/null; then
    if ! redis-cli ping &> /dev/null; then
        echo -e "${YELLOW}警告: Redis未运行，某些测试可能会跳过${NC}"
    fi
fi

# 安装测试依赖
echo -e "${YELLOW}安装测试依赖...${NC}"
go mod download

# 运行单元测试
echo -e "${GREEN}运行单元测试...${NC}"
go test ./test/unit -v -short

# 运行集成测试（需要Redis）
echo -e "${GREEN}运行集成测试...${NC}"
echo -e "${YELLOW}注意：集成测试需要Redis服务器${NC}"

# 检查Redis是否可用
if command -v redis-cli &> /dev/null && redis-cli ping &> /dev/null; then
    # 启动测试服务器
    echo "启动测试服务器..."
    go run ./cmd/server &
    SERVER_PID=$!
    
    # 等待服务器启动
    sleep 2
    
    # 运行集成测试
    go test ./test/integration -v
    
    # 停止测试服务器
    kill $SERVER_PID 2>/dev/null || true
else
    echo -e "${YELLOW}跳过集成测试：Redis不可用${NC}"
fi

# 代码覆盖率
echo -e "${GREEN}生成代码覆盖率报告...${NC}"
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html

echo -e "${GREEN}代码覆盖率报告已生成：coverage.html${NC}"

# 运行所有测试（简短模式）
echo -e "${GREEN}运行所有测试（简短模式）...${NC}"
go test ./... -short

echo "========================================="
echo -e "${GREEN}所有测试完成！${NC}"
echo "========================================="