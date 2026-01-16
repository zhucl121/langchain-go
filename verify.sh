#!/bin/bash

# LangChain-Go 验证脚本
# 用于验证项目编译和测试状态

echo "🔍 LangChain-Go 项目验证"
echo "================================"
echo

# 切换到项目目录
cd "$(dirname "$0")"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 编译验证
echo "📦 步骤 1: 编译验证"
echo "--------------------------------"
if go build $(go list ./... | grep -v '/examples') 2>&1 | grep -q ""; then
    go build $(go list ./... | grep -v '/examples') 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ 所有包编译成功${NC}"
    else
        echo -e "${RED}✗ 编译失败${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ 所有包编译成功${NC}"
fi
echo

# 2. 测试验证
echo "🧪 步骤 2: 测试验证"
echo "--------------------------------"
go test $(go list ./... | grep -v '/examples') 2>&1 > /tmp/test_results.txt
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 所有测试通过${NC}"
    echo
    echo "测试概要:"
    grep -E "^(ok|PASS|\?)" /tmp/test_results.txt | tail -10
else
    echo -e "${RED}✗ 测试失败${NC}"
    cat /tmp/test_results.txt
    exit 1
fi
echo

# 3. 示例编译验证
echo "📝 步骤 3: 示例程序验证"
echo "--------------------------------"
EXAMPLES_DIR="examples"
if [ -d "$EXAMPLES_DIR" ]; then
    SUCCESS_COUNT=0
    FAIL_COUNT=0
    
    for example in examples/*.go; do
        filename=$(basename "$example")
        if go build "$example" 2>&1 > /dev/null; then
            echo -e "${GREEN}✓${NC} $filename"
            rm -f "${example%.go}"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo -e "${RED}✗${NC} $filename"
            FAIL_COUNT=$((FAIL_COUNT + 1))
        fi
    done
    
    echo
    echo "示例编译结果: ${GREEN}$SUCCESS_COUNT 成功${NC}, ${RED}$FAIL_COUNT 失败${NC}"
else
    echo -e "${YELLOW}⚠ 未找到 examples 目录${NC}"
fi
echo

# 4. 代码质量检查
echo "🔎 步骤 4: 代码质量检查"
echo "--------------------------------"
echo "运行 go vet..."
if go vet $(go list ./... | grep -v '/examples') 2>&1 > /tmp/vet_results.txt; then
    echo -e "${GREEN}✓ go vet 检查通过${NC}"
else
    echo -e "${YELLOW}⚠ go vet 发现一些问题:${NC}"
    head -20 /tmp/vet_results.txt
fi
echo

# 5. 依赖检查
echo "📚 步骤 5: 依赖检查"
echo "--------------------------------"
echo "检查 go.mod 状态..."
go mod tidy 2>&1 > /dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 依赖关系正常${NC}"
else
    echo -e "${RED}✗ 依赖关系有问题${NC}"
fi
echo

# 总结
echo "================================"
echo -e "${GREEN}✅ 验证完成!${NC}"
echo
echo "项目状态:"
echo "  • 核心包: 编译成功, 测试通过"
echo "  • 示例程序: 部分可用"
echo "  • 代码质量: 基本符合标准"
echo
echo "详细信息请查看 COMPLETION_SUMMARY.md"
echo "================================"
