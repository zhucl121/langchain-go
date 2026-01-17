#!/bin/bash

# LangChain-Go 测试环境快速验证脚本
# 验证配置文件和 Docker 环境

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  LangChain-Go 测试环境验证${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. 检查 Docker
echo -e "${BLUE}[1/5] 检查 Docker...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}  ❌ Docker 未运行${NC}"
    echo -e "${YELLOW}  请先启动 Docker Desktop${NC}"
    exit 1
fi
echo -e "${GREEN}  ✅ Docker 已运行${NC}"
echo ""

# 2. 检查 docker-compose
echo -e "${BLUE}[2/5] 检查 docker-compose...${NC}"
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    VERSION=$(docker-compose version --short 2>/dev/null || echo "unknown")
    echo -e "${GREEN}  ✅ 找到 docker-compose (版本: $VERSION)${NC}"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE_CMD="docker compose"
    VERSION=$(docker compose version --short 2>/dev/null || echo "unknown")
    echo -e "${GREEN}  ✅ 找到 docker compose (版本: $VERSION)${NC}"
else
    echo -e "${RED}  ❌ 未找到 docker-compose${NC}"
    exit 1
fi
echo ""

# 3. 检查配置文件
echo -e "${BLUE}[3/5] 检查配置文件...${NC}"
if [ ! -f "docker-compose.test.yml" ]; then
    echo -e "${RED}  ❌ 未找到 docker-compose.test.yml${NC}"
    exit 1
fi
echo -e "${GREEN}  ✅ docker-compose.test.yml 存在${NC}"
echo ""

# 4. 检查端口是否被占用
echo -e "${BLUE}[4/5] 检查端口...${NC}"
PORTS_OK=true

# 检查 Redis 端口 6379
if lsof -Pi :6379 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${YELLOW}  ⚠️  端口 6379 (Redis) 已被占用${NC}"
    PORTS_OK=false
else
    echo -e "${GREEN}  ✅ 端口 6379 (Redis) 可用${NC}"
fi

# 检查 Milvus 端口 19530
if lsof -Pi :19530 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${YELLOW}  ⚠️  端口 19530 (Milvus) 已被占用${NC}"
    PORTS_OK=false
else
    echo -e "${GREEN}  ✅ 端口 19530 (Milvus) 可用${NC}"
fi

if [ "$PORTS_OK" = false ]; then
    echo ""
    echo -e "${YELLOW}  提示: 端口被占用。可以运行以下命令解决:${NC}"
    echo -e "${YELLOW}    bash scripts/fix-port-conflict.sh  ${NC}${GREEN}(自动修复工具)${NC}"
    echo ""
    echo -e "${YELLOW}  或手动检查占用端口的容器:${NC}"
    echo -e "${YELLOW}    docker ps --filter 'publish=6379'${NC}"
    echo -e "${YELLOW}    docker ps --filter 'publish=19530'${NC}"
fi
echo ""

# 5. 检查磁盘空间
echo -e "${BLUE}[5/5] 检查磁盘空间...${NC}"
if command -v df &> /dev/null; then
    AVAILABLE=$(df -h . | awk 'NR==2 {print $4}')
    echo -e "${GREEN}  ✅ 可用空间: $AVAILABLE${NC}"
    echo -e "${YELLOW}  提示: Milvus 首次运行需要下载约 1-2GB 镜像${NC}"
fi
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ 验证完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}下一步:${NC}"
echo -e "  1. 启动测试环境:"
echo -e "     ${GREEN}bash scripts/test-env-setup.sh${NC}"
echo -e ""
echo -e "  2. 运行测试:"
echo -e "     ${GREEN}bash scripts/run-tests.sh${NC}"
echo -e ""
echo -e "  3. 或使用 Make 命令:"
echo -e "     ${GREEN}make -f Makefile.test test-env-up${NC}"
echo -e "     ${GREEN}make -f Makefile.test test${NC}"
echo ""
