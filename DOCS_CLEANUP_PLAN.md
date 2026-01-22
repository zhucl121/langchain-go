# 文档整理计划

## 📋 整理目标

清理根目录下的文档，保持根目录简洁，将版本相关文档归档。

---

## 📂 保留在根目录的核心文档

这些是项目的核心文档，必须保留在根目录：

| 文件名 | 说明 | 大小 | 状态 |
|--------|------|------|------|
| `README.md` | 项目主页，第一印象 | 16K | ✅ 保留 |
| `CHANGELOG.md` | 完整变更日志 | 19K | ✅ 保留 |
| `CONTRIBUTING.md` | 贡献指南 | 10K | ✅ 保留 |
| `CODE_OF_CONDUCT.md` | 行为准则 | 2.6K | ✅ 保留 |
| `SECURITY.md` | 安全策略 | 1.7K | ✅ 保留 |
| `TESTING.md` | 测试指南 | 6.4K | ✅ 保留 |
| `QUICK_START.md` | 快速开始 | 9.4K | ✅ 保留 |
| `DOCUMENTATION_STRUCTURE.md` | 文档结构说明 | 7.4K | ✅ 保留 |
| `LICENSE` | 许可证 | - | ✅ 保留 |

### 当前版本文档
| `RELEASE_v0.6.0.sh` | v0.6.0 发布脚本 | 1.4K | ✅ 保留（当前版本）|
| `verify.sh` | 环境验证脚本 | 2.9K | ✅ 保留 |

**总计**: 10 个核心文档（~82K）

---

## 🗂️ 需要归档的文档

### 1. 移动到 `docs/releases/`

| 原路径 | 新路径 | 说明 | 大小 |
|--------|--------|------|------|
| `V0.6.0_RELEASE_COMPLETE.md` | `docs/releases/V0.6.0_RELEASE_COMPLETE.md` | v0.6.0 发布完成报告 | 7.4K |
| `CI_FIX_SUMMARY.md` | `docs/releases/CI_FIX_SUMMARY_v0.6.0.md` | v0.6.0 CI 修复总结 | 8.2K |

### 2. 移动到 `docs/releases/archive/` （旧版本）

| 原路径 | 新路径 | 说明 | 大小 |
|--------|--------|------|------|
| `V0.4.1_READY_TO_PUBLISH.md` | `docs/releases/archive/V0.4.1_READY_TO_PUBLISH.md` | v0.4.1 发布准备 | 8.2K |
| `V0.5.0_发布说明.md` | `docs/releases/archive/V0.5.0_发布说明.md` | v0.5.0 发布说明 | 5.0K |

### 3. 移动到 `scripts/archive/` （旧版本脚本）

| 原路径 | 新路径 | 说明 | 大小 |
|--------|--------|------|------|
| `release_v0.5.0.sh` | `scripts/archive/release_v0.5.0.sh` | v0.5.0 发布脚本 | 3.6K |

**归档文档总计**: 5 个文件（~32K）

---

## 📝 执行步骤

### Step 1: 移动当前版本文档到 docs/releases/
```bash
mv V0.6.0_RELEASE_COMPLETE.md docs/releases/
mv CI_FIX_SUMMARY.md docs/releases/CI_FIX_SUMMARY_v0.6.0.md
```

### Step 2: 归档旧版本文档
```bash
mv V0.4.1_READY_TO_PUBLISH.md docs/releases/archive/
mv V0.5.0_发布说明.md docs/releases/archive/
```

### Step 3: 归档旧版本脚本
```bash
mv release_v0.5.0.sh scripts/archive/
```

### Step 4: 更新相关文档索引
- 更新 `docs/releases/README.md`
- 更新 `DOCUMENTATION_STRUCTURE.md`

### Step 5: Git 提交
```bash
git add -A
git commit -m "docs: 整理根目录文档，归档旧版本文件"
```

---

## 📊 整理后的目录结构

### 根目录（简洁清爽）
```
/
├── README.md                          # 项目主页
├── CHANGELOG.md                       # 变更日志
├── CONTRIBUTING.md                    # 贡献指南
├── CODE_OF_CONDUCT.md                 # 行为准则
├── SECURITY.md                        # 安全策略
├── TESTING.md                         # 测试指南
├── QUICK_START.md                     # 快速开始
├── DOCUMENTATION_STRUCTURE.md         # 文档结构
├── LICENSE                            # 许可证
├── RELEASE_v0.6.0.sh                  # 当前版本发布脚本 ⭐
└── verify.sh                          # 验证脚本
```

### docs/releases/（版本发布文档）
```
docs/releases/
├── README.md                          # 发布历史索引
├── V0.6.0_RELEASE_COMPLETE.md         # v0.6.0 发布完成 ⭐ 新增
├── CI_FIX_SUMMARY_v0.6.0.md           # v0.6.0 CI 修复 ⭐ 新增
├── RELEASE_NOTES_v0.6.0.md            # v0.6.0 完整说明
├── GITHUB_RELEASE_v0.6.0.md           # v0.6.0 GitHub Release
├── RELEASE_CHECKLIST_v0.6.0.md        # v0.6.0 检查清单
├── ... (其他版本)
└── archive/                           # 旧版本归档 ⭐ 新增
    ├── V0.4.1_READY_TO_PUBLISH.md
    └── V0.5.0_发布说明.md
```

### scripts/（脚本文件）
```
scripts/
├── prepare-release.sh                 # 通用发布准备
├── ... (其他脚本)
└── archive/                           # 旧脚本归档 ⭐ 新增
    └── release_v0.5.0.sh
```

---

## ✅ 整理效果

### 优点

1. **根目录简洁**: 只保留核心文档（10 个）
2. **版本归档**: 旧版本文档统一归档到 `archive/`
3. **结构清晰**: 文档按类型和版本组织
4. **易于维护**: 新版本发布时只需更新 `RELEASE_vX.X.X.sh`

### 命名规范

- **核心文档**: 全大写（如 `README.md`）
- **版本文档**: `VX.X.X_*.md` 格式
- **脚本**: `*.sh` 小写下划线分隔
- **归档**: 放在 `archive/` 子目录

---

## 🎯 预期结果

执行完成后：
- ✅ 根目录只有 10-12 个核心文档
- ✅ 版本文档全部在 `docs/releases/`
- ✅ 旧版本归档到 `archive/`
- ✅ 文档结构清晰易懂
- ✅ 符合项目规范（.cursorrules）

---

**执行日期**: 2026-01-22  
**负责人**: LangChain-Go Team
