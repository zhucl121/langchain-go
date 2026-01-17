#!/bin/bash

# langchain-go 测试环境停止脚本
# 用于停止 Redis 和 Milvus 测试环境

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  停止测试环境${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 docker-compose 命令
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker compose"
else
    echo -e "${RED}❌ 未找到 docker-compose 命令${NC}"
    exit 1
fi

# 停止服务
echo -e "${YELLOW}⏳ 停止服务...${NC}"
$DOCKER_COMPOSE_CMD -f docker-compose.test.yml down

echo ""
echo -e "${GREEN}✅ 测试环境已停止${NC}"
echo ""
echo -e "${BLUE}如需删除数据卷:${NC}"
echo -e "  ${YELLOW}$DOCKER_COMPOSE_CMD -f docker-compose.test.yml down -v${NC}"
echo ""
