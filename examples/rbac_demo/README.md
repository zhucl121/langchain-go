# RBAC & 多租户演示

这个示例演示了 LangChain-Go v0.6.0 的企业级功能：

- RBAC (基于角色的访问控制)
- 多租户管理
- 配额管理
- 权限检查

## 功能演示

1. **创建租户** - 创建一个演示租户并设置配额
2. **添加成员** - 向租户添加用户并分配角色
3. **权限检查** - 测试不同角色的权限
4. **配额管理** - 检查资源使用量和配额
5. **自定义角色** - 创建自定义角色并分配
6. **查看权限** - 查看用户的角色和权限

## 运行示例

```bash
cd examples/rbac_demo
go run main.go
```

## 预期输出

```
=== LangChain-Go v0.6.0 - RBAC & 多租户演示 ===

>>> 创建租户
✓ 租户创建成功: 演示公司 (tenant-demo)

>>> 添加租户成员
✓ 用户 user-alice 添加为 developer
✓ 用户 user-bob 添加为 viewer
✓ 用户 user-charlie 添加为 data-scientist

>>> 租户信息
租户: 演示公司
成员数量: 3
配额:
  - 最大 Agent 数: 100
  - 最大向量存储: 10
  - 最大文档数: 10000

>>> 权限检查测试
✓ 允许: user-alice 对 agent 执行 read 操作
✓ 允许: user-alice 对 agent 执行 write 操作
✗ 拒绝: user-alice 对 agent 执行 delete 操作
✓ 允许: user-bob 对 agent 执行 read 操作
✗ 拒绝: user-bob 对 agent 执行 write 操作
✓ 允许: user-charlie 对 model 执行 execute 操作

>>> 配额检查测试
agent: Allowed: 50/100 (remaining: 50)
document: Allowed: 5000/10000 (remaining: 5000)
api_call: Allowed: 0/100000 (remaining: 100000)

>>> 创建自定义角色
✓ 自定义角色创建成功: 机器学习工程师
  权限数量: 3
✓ 用户 user-david 已分配角色: 机器学习工程师

>>> 查看用户权限
用户 user-alice 的角色:
  - 开发者 (developer)
    权限: 6 个

>>> 内置角色
- 系统管理员 (system-admin)
  描述: 系统管理员，拥有所有权限
  权限: 1 个

- 租户管理员 (tenant-admin)
  描述: 租户管理员，租户内所有权限
  权限: 1 个

- 开发者 (developer)
  描述: 开发者，拥有读写和执行权限
  权限: 6 个

- 查看者 (viewer)
  描述: 查看者，仅拥有只读权限
  权限: 1 个

- 数据科学家 (data-scientist)
  描述: 数据科学家，拥有模型和数据操作权限
  权限: 4 个

- 运维人员 (operator)
  描述: 运维人员，拥有监控和管理权限
  权限: 3 个

=== 演示完成 ===
```

## 核心概念

### RBAC (基于角色的访问控制)

- **角色 (Role)**: 权限的集合
- **权限 (Permission)**: 对资源的操作许可
- **用户 (User)**: 可以分配多个角色
- **范围 (Scope)**: 全局、租户、资源三个级别

### 多租户

- **租户 (Tenant)**: 资源和数据隔离的基本单位
- **配额 (Quota)**: 限制租户的资源使用
- **成员 (Member)**: 租户内的用户及其角色
- **隔离 (Isolation)**: 确保租户数据完全隔离

## 相关文档

- [v0.6.0 用户指南](../../docs/V0.6.0_USER_GUIDE.md)
- [v0.6.0 安全指南](../../docs/V0.6.0_SECURITY_GUIDE.md)
- [v0.6.0 API 参考](../../docs/V0.6.0_API_REFERENCE.md)
