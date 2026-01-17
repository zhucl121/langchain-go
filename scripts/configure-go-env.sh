#!/bin/bash

# LangChain-Go 环境配置脚本
# 自动配置 Go 环境变量

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  配置 Go 环境${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检测 shell
if [ -n "$ZSH_VERSION" ]; then
    SHELL_CONFIG="$HOME/.zshrc"
    SHELL_NAME="zsh"
elif [ -n "$BASH_VERSION" ]; then
    if [ -f "$HOME/.bash_profile" ]; then
        SHELL_CONFIG="$HOME/.bash_profile"
    else
        SHELL_CONFIG="$HOME/.bashrc"
    fi
    SHELL_NAME="bash"
else
    echo -e "${YELLOW}⚠️  未能检测 shell 类型，请手动配置${NC}"
    exit 1
fi

echo -e "${BLUE}检测到 shell: ${GREEN}$SHELL_NAME${NC}"
echo -e "${BLUE}配置文件: ${GREEN}$SHELL_CONFIG${NC}"
echo ""

# 检查是否已配置
if grep -q "/usr/local/go/bin" "$SHELL_CONFIG" 2>/dev/null; then
    echo -e "${GREEN}✅ PATH 已包含 /usr/local/go/bin${NC}"
    echo ""
else
    echo -e "${YELLOW}添加 Go 到 PATH...${NC}"
    echo "" >> "$SHELL_CONFIG"
    echo "# Go 1.25.6" >> "$SHELL_CONFIG"
    echo 'export PATH="/usr/local/go/bin:$PATH"' >> "$SHELL_CONFIG"
    echo -e "${GREEN}✅ 已添加到 $SHELL_CONFIG${NC}"
    echo ""
fi

# 验证
echo -e "${BLUE}验证 Go 安装...${NC}"
/usr/local/go/bin/go version
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ 配置完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}请运行以下命令使配置生效:${NC}"
echo -e "  ${GREEN}source $SHELL_CONFIG${NC}"
echo ""
echo -e "或者重新打开终端"
echo ""
echo -e "${BLUE}验证配置:${NC}"
echo -e "  ${GREEN}go version${NC}"
echo ""
