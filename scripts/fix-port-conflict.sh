#!/bin/bash

# 端口冲突快速修复脚本

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  端口冲突修复工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查端口占用
echo -e "${BLUE}检查端口占用情况...${NC}"
echo ""

# 检查 6379 (Redis)
if lsof -Pi :6379 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${YELLOW}⚠️  端口 6379 (Redis) 已被占用${NC}"
    REDIS_OCCUPIED=true
else
    echo -e "${GREEN}✅ 端口 6379 (Redis) 可用${NC}"
    REDIS_OCCUPIED=false
fi

# 检查 19530 (Milvus)
if lsof -Pi :19530 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo -e "${YELLOW}⚠️  端口 19530 (Milvus) 已被占用${NC}"
    MILVUS_OCCUPIED=true
else
    echo -e "${GREEN}✅ 端口 19530 (Milvus) 可用${NC}"
    MILVUS_OCCUPIED=false
fi

echo ""

if [ "$REDIS_OCCUPIED" = false ] && [ "$MILVUS_OCCUPIED" = false ]; then
    echo -e "${GREEN}✅ 所有端口都可用，无需修复${NC}"
    exit 0
fi

# 查找占用端口的 Docker 容器
echo -e "${BLUE}查找占用端口的容器...${NC}"
echo ""

if [ "$REDIS_OCCUPIED" = true ]; then
    echo -e "${YELLOW}Redis 端口 (6379) 被以下容器占用:${NC}"
    docker ps --filter "publish=6379" --format "  • {{.Names}} ({{.Image}})" 2>/dev/null || echo "  无法获取容器信息"
    echo ""
fi

if [ "$MILVUS_OCCUPIED" = true ]; then
    echo -e "${YELLOW}Milvus 端口 (19530) 被以下容器占用:${NC}"
    docker ps --filter "publish=19530" --format "  • {{.Names}} ({{.Image}})" 2>/dev/null || echo "  无法获取容器信息"
    echo ""
fi

# 提供解决方案
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  解决方案${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

echo -e "${CYAN}方案 1: 停止占用端口的容器 (推荐)${NC}"
echo ""

if [ "$REDIS_OCCUPIED" = true ]; then
    REDIS_CONTAINER=$(docker ps --filter "publish=6379" --format "{{.Names}}" 2>/dev/null | head -1)
    if [ -n "$REDIS_CONTAINER" ]; then
        echo -e "  停止 Redis 容器:"
        echo -e "  ${GREEN}docker stop $REDIS_CONTAINER${NC}"
        echo ""
    fi
fi

if [ "$MILVUS_OCCUPIED" = true ]; then
    MILVUS_CONTAINER=$(docker ps --filter "publish=19530" --format "{{.Names}}" 2>/dev/null | head -1)
    if [ -n "$MILVUS_CONTAINER" ]; then
        echo -e "  停止 Milvus 容器:"
        echo -e "  ${GREEN}docker stop $MILVUS_CONTAINER${NC}"
        echo ""
    fi
fi

echo -e "${CYAN}方案 2: 使用不同的端口${NC}"
echo ""
echo -e "  修改 docker-compose.test.yml 中的端口映射:"
if [ "$REDIS_OCCUPIED" = true ]; then
    echo -e "  • Redis: ${YELLOW}6380:6379${NC} (使用 6380 端口)"
fi
if [ "$MILVUS_OCCUPIED" = true ]; then
    echo -e "  • Milvus: ${YELLOW}19531:19530${NC} (使用 19531 端口)"
fi
echo ""

echo -e "${CYAN}方案 3: 停止测试环境的容器${NC}"
echo ""
echo -e "  如果测试容器已创建但未启动:"
echo -e "  ${GREEN}docker compose -f docker-compose.test.yml down${NC}"
echo ""

# 提供自动修复选项
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  自动修复${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if [ "$REDIS_OCCUPIED" = true ] && [ -n "$REDIS_CONTAINER" ]; then
    read -p "是否自动停止 $REDIS_CONTAINER？(y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}停止 $REDIS_CONTAINER...${NC}"
        docker stop "$REDIS_CONTAINER"
        echo -e "${GREEN}✅ 已停止 $REDIS_CONTAINER${NC}"
        echo ""
    fi
fi

if [ "$MILVUS_OCCUPIED" = true ] && [ -n "$MILVUS_CONTAINER" ]; then
    read -p "是否自动停止 $MILVUS_CONTAINER？(y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}停止 $MILVUS_CONTAINER...${NC}"
        docker stop "$MILVUS_CONTAINER"
        echo -e "${GREEN}✅ 已停止 $MILVUS_CONTAINER${NC}"
        echo ""
    fi
fi

echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "现在可以尝试启动测试环境:"
echo -e "  ${GREEN}make -f Makefile.test test-env-up${NC}"
echo ""
