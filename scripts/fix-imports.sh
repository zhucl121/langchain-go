#!/bin/bash

# 批量替换 langchain-go 导入路径为 github.com/zhucl121/langchain-go

set -e

echo "=== 更新 langchain-go 导入路径 ==="
echo ""

cd /Users/yunyuexingsheng/Documents/worksapce/随笔/langchain-go

# 统计需要替换的文件数
total=$(find . -name "*.go" -type f ! -path "./vendor/*" -exec grep -l "\"langchain-go/" {} \; | wc -l)
echo "找到 $total 个文件需要更新"
echo ""

# 执行替换
find . -name "*.go" -type f ! -path "./vendor/*" -print0 | while IFS= read -r -d '' file; do
    if grep -q "\"langchain-go/" "$file"; then
        echo "更新: $file"
        sed -i '' 's|"langchain-go/|"github.com/zhucl121/langchain-go/|g' "$file"
    fi
done

echo ""
echo "✓ 所有文件已更新"
echo ""

# 验证
echo "验证更新..."
remaining=$(find . -name "*.go" -type f ! -path "./vendor/*" -exec grep -l "\"langchain-go/" {} \; | wc -l)
echo "剩余未更新的文件: $remaining"

if [ "$remaining" -eq 0 ]; then
    echo "✓ 所有导入路径已成功更新"
else
    echo "⚠ 还有文件未更新，请检查"
fi
