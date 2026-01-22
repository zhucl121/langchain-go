package rbac

import (
	"time"
)

var (
	// RoleSystemAdmin 系统管理员 - 所有权限
	RoleSystemAdmin = &Role{
		ID:          "system-admin",
		Name:        "系统管理员",
		Description: "系统管理员，拥有所有权限",
		Permissions: []*Permission{
			{Resource: "*", Actions: []string{"*"}, Scope: ScopeGlobal},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// RoleTenantAdmin 租户管理员 - 租户内所有权限
	RoleTenantAdmin = &Role{
		ID:          "tenant-admin",
		Name:        "租户管理员",
		Description: "租户管理员，租户内所有权限",
		Permissions: []*Permission{
			{Resource: "*", Actions: []string{"*"}, Scope: ScopeTenant},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// RoleDeveloper 开发者 - 读写和执行权限
	RoleDeveloper = &Role{
		ID:          "developer",
		Name:        "开发者",
		Description: "开发者，拥有读写和执行权限",
		Permissions: []*Permission{
			{Resource: "agent", Actions: []string{"read", "write", "execute"}, Scope: ScopeTenant},
			{Resource: "model", Actions: []string{"read", "execute"}, Scope: ScopeTenant},
			{Resource: "vectorstore", Actions: []string{"read", "write"}, Scope: ScopeTenant},
			{Resource: "document", Actions: []string{"read", "write"}, Scope: ScopeTenant},
			{Resource: "tool", Actions: []string{"read", "execute"}, Scope: ScopeTenant},
			{Resource: "memory", Actions: []string{"read", "write"}, Scope: ScopeTenant},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// RoleViewer 查看者 - 只读权限
	RoleViewer = &Role{
		ID:          "viewer",
		Name:        "查看者",
		Description: "查看者，仅拥有只读权限",
		Permissions: []*Permission{
			{Resource: "*", Actions: []string{"read"}, Scope: ScopeTenant},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// RoleDataScientist 数据科学家 - 模型和数据权限
	RoleDataScientist = &Role{
		ID:          "data-scientist",
		Name:        "数据科学家",
		Description: "数据科学家，拥有模型和数据操作权限",
		Permissions: []*Permission{
			{Resource: "model", Actions: []string{"read", "execute"}, Scope: ScopeTenant},
			{Resource: "vectorstore", Actions: []string{"read", "write"}, Scope: ScopeTenant},
			{Resource: "document", Actions: []string{"read", "write"}, Scope: ScopeTenant},
			{Resource: "agent", Actions: []string{"read"}, Scope: ScopeTenant},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// RoleOperator 运维人员 - 监控和管理权限
	RoleOperator = &Role{
		ID:          "operator",
		Name:        "运维人员",
		Description: "运维人员，拥有监控和管理权限",
		Permissions: []*Permission{
			{Resource: "*", Actions: []string{"read"}, Scope: ScopeTenant},
			{Resource: "monitoring", Actions: []string{"*"}, Scope: ScopeTenant},
			{Resource: "logs", Actions: []string{"read"}, Scope: ScopeTenant},
		},
		IsSystem:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
)

// GetBuiltinRoles 获取所有内置角色
func GetBuiltinRoles() []*Role {
	return []*Role{
		RoleSystemAdmin,
		RoleTenantAdmin,
		RoleDeveloper,
		RoleViewer,
		RoleDataScientist,
		RoleOperator,
	}
}

// IsBuiltinRole 检查是否为内置角色
func IsBuiltinRole(roleID string) bool {
	builtinRoleIDs := []string{
		"system-admin",
		"tenant-admin",
		"developer",
		"viewer",
		"data-scientist",
		"operator",
	}

	for _, id := range builtinRoleIDs {
		if id == roleID {
			return true
		}
	}
	return false
}
