package node

import (
	"testing"
)

func TestNodeFilter_Match(t *testing.T) {
	node := &Node{
		ID:      "node-1",
		Name:    "worker-1",
		Address: "192.168.1.10",
		Port:    8080,
		Status:  StatusOnline,
		Roles:   []NodeRole{RoleWorker},
		Region:  "us-east-1",
		Zone:    "us-east-1a",
		Capacity: Capacity{
			MaxConnections: 1000,
			MaxQPS:         500,
			MaxMemoryMB:    4096,
		},
		Load: Load{
			CurrentConnections: 500,
			CPUUsagePercent:    50.0,
		},
		Metadata: map[string]string{
			"env": "production",
		},
	}

	tests := []struct {
		name   string
		filter *NodeFilter
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: &NodeFilter{},
			want:   true,
		},
		{
			name: "status filter matches",
			filter: &NodeFilter{
				Status: []NodeStatus{StatusOnline},
			},
			want: true,
		},
		{
			name: "status filter no match",
			filter: &NodeFilter{
				Status: []NodeStatus{StatusOffline},
			},
			want: false,
		},
		{
			name: "role filter matches",
			filter: &NodeFilter{
				Roles: []NodeRole{RoleWorker},
			},
			want: true,
		},
		{
			name: "role filter no match",
			filter: &NodeFilter{
				Roles: []NodeRole{RoleMaster},
			},
			want: false,
		},
		{
			name: "region filter matches",
			filter: &NodeFilter{
				Region: "us-east-1",
			},
			want: true,
		},
		{
			name: "region filter no match",
			filter: &NodeFilter{
				Region: "us-west-2",
			},
			want: false,
		},
		{
			name: "min capacity matches",
			filter: &NodeFilter{
				MinCapacity: &Capacity{
					MaxConnections: 500,
				},
			},
			want: true,
		},
		{
			name: "min capacity no match",
			filter: &NodeFilter{
				MinCapacity: &Capacity{
					MaxConnections: 2000,
				},
			},
			want: false,
		},
		{
			name: "max load matches",
			filter: &NodeFilter{
				MaxLoad: &Load{
					CurrentConnections: 600,
					CPUUsagePercent:    60.0,
				},
			},
			want: true,
		},
		{
			name: "max load no match",
			filter: &NodeFilter{
				MaxLoad: &Load{
					CurrentConnections: 400,
				},
			},
			want: false,
		},
		{
			name: "tags match",
			filter: &NodeFilter{
				Tags: map[string]string{
					"env": "production",
				},
			},
			want: true,
		},
		{
			name: "tags no match",
			filter: &NodeFilter{
				Tags: map[string]string{
					"env": "staging",
				},
			},
			want: false,
		},
		{
			name: "healthy only matches",
			filter: &NodeFilter{
				HealthyOnly: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Match(node); got != tt.want {
				t.Errorf("NodeFilter.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeFilter_MatchAny(t *testing.T) {
	nodes := []*Node{
		{
			ID:     "node-1",
			Status: StatusOnline,
			Roles:  []NodeRole{RoleWorker},
		},
		{
			ID:     "node-2",
			Status: StatusOffline,
			Roles:  []NodeRole{RoleMaster},
		},
		{
			ID:     "node-3",
			Status: StatusOnline,
			Roles:  []NodeRole{RoleCache},
		},
	}

	filter := &NodeFilter{
		Status: []NodeStatus{StatusOnline},
	}

	matched := filter.MatchAny(nodes)
	if len(matched) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matched))
	}
}

func TestNodeFilter_Chaining(t *testing.T) {
	filter := NewNodeFilter().
		WithStatus(StatusOnline).
		WithRoles(RoleWorker).
		WithRegion("us-east-1").
		WithZone("us-east-1a").
		WithTag("env", "production").
		WithHealthyOnly()

	if len(filter.Status) != 1 || filter.Status[0] != StatusOnline {
		t.Error("Status not set correctly")
	}
	if len(filter.Roles) != 1 || filter.Roles[0] != RoleWorker {
		t.Error("Roles not set correctly")
	}
	if filter.Region != "us-east-1" {
		t.Error("Region not set correctly")
	}
	if filter.Zone != "us-east-1a" {
		t.Error("Zone not set correctly")
	}
	if filter.Tags["env"] != "production" {
		t.Error("Tags not set correctly")
	}
	if !filter.HealthyOnly {
		t.Error("HealthyOnly not set correctly")
	}
}
