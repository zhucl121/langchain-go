#!/bin/bash

# langchain-go 测试环境启动脚本
# 用于启动 Redis 和 Milvus 2.6.1 测试环境

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  LangChain-Go 测试环境启动${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker 未运行，请先启动 Docker${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker 已运行${NC}"
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

echo -e "${GREEN}✅ 使用命令: $DOCKER_COMPOSE_CMD${NC}"
echo ""

# 停止并删除旧容器
echo -e "${YELLOW}⏳ 清理旧容器...${NC}"
$DOCKER_COMPOSE_CMD -f docker-compose.test.yml down -v 2>/dev/null || true
echo ""

# 启动服务
echo -e "${YELLOW}⏳ 启动测试服务...${NC}"
$DOCKER_COMPOSE_CMD -f docker-compose.test.yml up -d

echo ""
echo -e "${YELLOW}⏳ 等待服务就绪...${NC}"

# 等待 Redis
echo -n "等待 Redis... "
for i in {1..30}; do
    if docker exec langchain-go-redis redis-cli -a redis123 ping 2>/dev/null | grep -q PONG; then
        echo -e "${GREEN}✅ Redis 就绪${NC}"
        break
    fi
    if [ $i -eq 30 ]; then
        echo -e "${RED}❌ Redis 启动超时${NC}"
        exit 1
    fi
    sleep 1
done

# 等待 Milvus
echo -n "等待 Milvus... "
for i in {1..60}; do
    if curl -s http://localhost:9091/healthz 2>/dev/null | grep -q "OK"; then
        echo -e "${GREEN}✅ Milvus 就绪${NC}"
        break
    fi
    if [ $i -eq 60 ]; then
        echo -e "${RED}❌ Milvus 启动超时${NC}"
        echo -e "${YELLOW}提示: Milvus 首次启动可能需要更长时间，请查看日志:${NC}"
        echo -e "${YELLOW}  $DOCKER_COMPOSE_CMD -f docker-compose.test.yml logs milvus${NC}"
        exit 1
    fi
    sleep 2
done

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ 测试环境启动成功！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${BLUE}服务信息:${NC}"
echo -e "  • Redis:  ${GREEN}localhost:6379${NC} (密码: redis123)"
echo -e "  • Milvus: ${GREEN}localhost:19530${NC}"
echo ""
echo -e "${BLUE}常用命令:${NC}"
echo -e "  • 查看日志: ${YELLOW}$DOCKER_COMPOSE_CMD -f docker-compose.test.yml logs -f [service]${NC}"
echo -e "  • 停止服务: ${YELLOW}$DOCKER_COMPOSE_CMD -f docker-compose.test.yml down${NC}"
echo -e "  • 重启服务: ${YELLOW}$DOCKER_COMPOSE_CMD -f docker-compose.test.yml restart${NC}"
echo ""
echo -e "${BLUE}运行测试:${NC}"
echo -e "  • 全部测试: ${YELLOW}go test ./...${NC}"
echo -e "  • Redis测试: ${YELLOW}go test ./core/cache -v${NC}"
echo -e "  • Milvus测试: ${YELLOW}go test ./retrieval/vectorstores -v${NC}"
echo ""
