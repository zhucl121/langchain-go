#!/bin/bash

# 批量替换 GitHub 仓库地址
# 从 zhuchenglong 更新为 zhucl121

set -e

cd "$(dirname "$0")/.."

echo "🔄 批量更新 GitHub 仓库地址..."
echo "   旧地址: github.com/zhuchenglong/langchain-go"
echo "   新地址: github.com/zhucl121/langchain-go"
echo ""

# 统计需要更新的文件
TOTAL_FILES=$(grep -r "zhuchenglong" --include="*.md" --include="*.go" . 2>/dev/null | cut -d: -f1 | sort -u | wc -l)
echo "📊 发现 $TOTAL_FILES 个文件需要更新"
echo ""

# 备份文件
echo "💾 创建备份..."
BACKUP_DIR=".repo_update_backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
find . -type f \( -name "*.md" -o -name "*.go" \) -exec grep -l "zhuchenglong" {} \; 2>/dev/null | \
  while read file; do
    mkdir -p "$BACKUP_DIR/$(dirname "$file")"
    cp "$file" "$BACKUP_DIR/$file"
  done
echo "✅ 备份完成: $BACKUP_DIR"
echo ""

# 执行替换
echo "🔄 开始替换..."

# macOS 和 Linux 兼容的 sed 命令
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS
  find . -type f \( -name "*.md" -o -name "*.go" \) -exec sed -i '' 's/zhuchenglong\/langchain-go/zhucl121\/langchain-go/g' {} + 2>/dev/null
else
  # Linux
  find . -type f \( -name "*.md" -o -name "*.go" \) -exec sed -i 's/zhuchenglong\/langchain-go/zhucl121\/langchain-go/g' {} + 2>/dev/null
fi

echo "✅ 替换完成"
echo ""

# 验证结果
REMAINING=$(grep -r "zhuchenglong" --include="*.md" --include="*.go" . 2>/dev/null | wc -l)
echo "📊 验证结果:"
echo "   剩余未替换: $REMAINING 处"

if [ "$REMAINING" -eq 0 ]; then
  echo "   ✅ 所有引用已成功更新！"
else
  echo "   ⚠️  还有一些文件未更新，请手动检查："
  grep -r "zhuchenglong" --include="*.md" --include="*.go" . 2>/dev/null | head -10
fi

echo ""
echo "🎉 更新完成！"
echo ""
echo "下一步："
echo "  1. 检查更改: git diff"
echo "  2. 提交更改: git add . && git commit -m 'chore: 更新 GitHub 仓库地址为 zhucl121'"
echo "  3. 推送到远程: git push origin main"
