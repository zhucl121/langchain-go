#!/bin/bash

# LangChain-Go v0.6.0 发布脚本
# 执行时间: 2026-01-22

echo "🚀 LangChain-Go v0.6.0 发布脚本"
echo "================================"
echo ""

# 1. 检查当前分支
echo "📌 1. 检查当前分支..."
git branch --show-current
echo ""

# 2. 检查本地 tag
echo "🏷️  2. 检查本地 tag..."
git tag -l v0.6.0
if [ $? -eq 0 ]; then
    echo "✅ Tag v0.6.0 已存在"
else
    echo "❌ Tag v0.6.0 不存在，请先创建 tag"
    exit 1
fi
echo ""

# 3. 推送到远程
echo "📤 3. 推送代码到远程..."
git push origin main
if [ $? -ne 0 ]; then
    echo "❌ 推送代码失败"
    exit 1
fi
echo "✅ 代码推送成功"
echo ""

# 4. 推送 tag
echo "🏷️  4. 推送 tag 到远程..."
git push origin v0.6.0
if [ $? -ne 0 ]; then
    echo "❌ 推送 tag 失败"
    exit 1
fi
echo "✅ Tag 推送成功"
echo ""

# 5. 验证远程 tag
echo "🔍 5. 验证远程 tag..."
git ls-remote --tags origin | grep v0.6.0
echo ""

# 6. 显示 GitHub Release 链接
echo "✅ v0.6.0 发布成功！"
echo ""
echo "📝 下一步："
echo "1. 访问 GitHub Release 页面创建发布："
echo "   https://github.com/zhucl121/langchain-go/releases/new"
echo ""
echo "2. 选择 tag: v0.6.0"
echo ""
echo "3. 标题: v0.6.0 - 企业级安全完整版"
echo ""
echo "4. 描述内容请参考："
echo "   docs/V0.6.0_COMPLETION_SUMMARY.md"
echo ""
echo "🎉 发布完成！"
