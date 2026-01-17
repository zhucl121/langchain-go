#!/bin/bash

# langchain-go 测试运行脚本
# 自动启动环境并运行测试

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  LangChain-Go 测试运行${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# 检查测试环境是否已启动
if ! docker ps | grep -q langchain-go-redis; then
    echo -e "${YELLOW}⏳ 测试环境未启动，正在启动...${NC}"
    bash "$SCRIPT_DIR/test-env-setup.sh"
    echo ""
else
    echo -e "${GREEN}✅ 测试环境已在运行${NC}"
    echo ""
fi

# 解析参数
TEST_PACKAGE="${1:-.}/..."
TEST_ARGS="${@:2}"

echo -e "${BLUE}运行测试:${NC}"
echo -e "  包: ${GREEN}$TEST_PACKAGE${NC}"
if [ -n "$TEST_ARGS" ]; then
    echo -e "  参数: ${GREEN}$TEST_ARGS${NC}"
fi
echo ""

# 运行测试
echo -e "${YELLOW}⏳ 开始测试...${NC}"
echo ""

if [ -n "$TEST_ARGS" ]; then
    go test "$TEST_PACKAGE" $TEST_ARGS
else
    go test "$TEST_PACKAGE" -v
fi

TEST_EXIT_CODE=$?

echo ""
if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ 测试全部通过！${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}❌ 测试失败 (退出码: $TEST_EXIT_CODE)${NC}"
    echo -e "${RED}========================================${NC}"
fi

exit $TEST_EXIT_CODE
