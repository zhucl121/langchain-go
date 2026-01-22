package node

import (
	"testing"
	"time"
)

func TestNode_IsHealthy(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want bool
	}{
		{
			name: "healthy node",
			node: &Node{
				Status: StatusOnline,
				Capacity: Capacity{
					MaxConnections: 1000,
					MaxMemoryMB:    4096,
				},
				Load: Load{
					CurrentConnections: 500,
					MemoryUsageMB:      2048,
					CPUUsagePercent:    50.0,
				},
			},
			want: true,
		},
		{
			name: "offline node",
			node: &Node{
				Status: StatusOffline,
			},
			want: false,
		},
		{
			name: "connections at capacity",
			node: &Node{
				Status: StatusOnline,
				Capacity: Capacity{
					MaxConnections: 1000,
				},
				Load: Load{
					CurrentConnections: 1000,
				},
			},
			want: false,
		},
		{
			name: "memory at capacity",
			node: &Node{
				Status: StatusOnline,
				Capacity: Capacity{
					MaxMemoryMB: 4096,
				},
				Load: Load{
					MemoryUsageMB: 4096,
				},
			},
			want: false,
		},
		{
			name: "high CPU usage",
			node: &Node{
				Status: StatusOnline,
				Load: Load{
					CPUUsagePercent: 96.0,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsHealthy(); got != tt.want {
				t.Errorf("Node.IsHealthy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_GetLoadPercent(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want float64
	}{
		{
			name: "50% load",
			node: &Node{
				Capacity: Capacity{MaxConnections: 1000},
				Load:     Load{CurrentConnections: 500},
			},
			want: 50.0,
		},
		{
			name: "100% load",
			node: &Node{
				Capacity: Capacity{MaxConnections: 1000},
				Load:     Load{CurrentConnections: 1000},
			},
			want: 100.0,
		},
		{
			name: "zero capacity",
			node: &Node{
				Capacity: Capacity{MaxConnections: 0},
				Load:     Load{CurrentConnections: 100},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.GetLoadPercent(); got != tt.want {
				t.Errorf("Node.GetLoadPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNode_HasRole(t *testing.T) {
	node := &Node{
		Roles: []NodeRole{RoleWorker, RoleCache},
	}

	tests := []struct {
		name string
		role NodeRole
		want bool
	}{
		{"has worker role", RoleWorker, true},
		{"has cache role", RoleCache, true},
		{"no master role", RoleMaster, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := node.HasRole(tt.role); got != tt.want {
				t.Errorf("Node.HasRole(%v) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestNode_GetEndpoint(t *testing.T) {
	node := &Node{
		Address: "192.168.1.10",
		Port:    8080,
	}

	want := "192.168.1.10:8080"
	if got := node.GetEndpoint(); got != want {
		t.Errorf("Node.GetEndpoint() = %v, want %v", got, want)
	}
}

func TestNode_GetURL(t *testing.T) {
	node := &Node{
		Address: "192.168.1.10",
		Port:    8080,
	}

	tests := []struct {
		name   string
		scheme string
		want   string
	}{
		{"default http", "", "http://192.168.1.10:8080"},
		{"https", "https", "https://192.168.1.10:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := node.GetURL(tt.scheme); got != tt.want {
				t.Errorf("Node.GetURL(%v) = %v, want %v", tt.scheme, got, tt.want)
			}
		})
	}
}

func TestNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    *Node
		wantErr bool
	}{
		{
			name: "valid node",
			node: &Node{
				ID:      "node-1",
				Name:    "worker-1",
				Address: "192.168.1.10",
				Port:    8080,
				Roles:   []NodeRole{RoleWorker},
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			node: &Node{
				Name:    "worker-1",
				Address: "192.168.1.10",
				Port:    8080,
				Roles:   []NodeRole{RoleWorker},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			node: &Node{
				ID:      "node-1",
				Address: "192.168.1.10",
				Port:    8080,
				Roles:   []NodeRole{RoleWorker},
			},
			wantErr: true,
		},
		{
			name: "missing address",
			node: &Node{
				ID:    "node-1",
				Name:  "worker-1",
				Port:  8080,
				Roles: []NodeRole{RoleWorker},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			node: &Node{
				ID:      "node-1",
				Name:    "worker-1",
				Address: "192.168.1.10",
				Port:    70000,
				Roles:   []NodeRole{RoleWorker},
			},
			wantErr: true,
		},
		{
			name: "no roles",
			node: &Node{
				ID:      "node-1",
				Name:    "worker-1",
				Address: "192.168.1.10",
				Port:    8080,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Node.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNode_Clone(t *testing.T) {
	original := &Node{
		ID:      "node-1",
		Name:    "worker-1",
		Address: "192.168.1.10",
		Port:    8080,
		Status:  StatusOnline,
		Roles:   []NodeRole{RoleWorker, RoleCache},
		Metadata: map[string]string{
			"env": "production",
		},
		RegisterAt: time.Now(),
		LastSeen:   time.Now(),
	}

	clone := original.Clone()

	// 检查值相等
	if clone.ID != original.ID {
		t.Errorf("Clone ID mismatch")
	}

	// 修改 clone 不应影响 original
	clone.Roles[0] = RoleMaster
	if original.Roles[0] == RoleMaster {
		t.Errorf("Modifying clone affected original")
	}

	clone.Metadata["env"] = "staging"
	if original.Metadata["env"] == "staging" {
		t.Errorf("Modifying clone metadata affected original")
	}
}

func TestNodeStatus_IsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		status NodeStatus
		want   bool
	}{
		{"online", StatusOnline, true},
		{"busy", StatusBusy, true},
		{"offline", StatusOffline, false},
		{"draining", StatusDraining, false},
		{"maintenance", StatusMaintenance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsAvailable(); got != tt.want {
				t.Errorf("NodeStatus.IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
