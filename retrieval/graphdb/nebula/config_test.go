package nebula_test

import (
	"testing"

	"github.com/zhucl121/langchain-go/retrieval/graphdb/nebula"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  nebula.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: nebula.Config{
				Addresses: []string{"127.0.0.1:9669"},
				Username:  "root",
				Password:  "nebula",
				Space:     "test",
			},
			wantErr: false,
		},
		{
			name: "missing addresses",
			config: nebula.Config{
				Username: "root",
				Password: "nebula",
				Space:    "test",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: nebula.Config{
				Addresses: []string{"127.0.0.1:9669"},
				Password:  "nebula",
				Space:     "test",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: nebula.Config{
				Addresses: []string{"127.0.0.1:9669"},
				Username:  "root",
				Space:     "test",
			},
			wantErr: true,
		},
		{
			name: "missing space",
			config: nebula.Config{
				Addresses: []string{"127.0.0.1:9669"},
				Username:  "root",
				Password:  "nebula",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := nebula.DefaultConfig()

	if len(config.Addresses) == 0 {
		t.Error("DefaultConfig should have addresses")
	}

	if config.Username == "" {
		t.Error("DefaultConfig should have username")
	}

	if config.Password == "" {
		t.Error("DefaultConfig should have password")
	}

	if config.Space == "" {
		t.Error("DefaultConfig should have space")
	}

	if config.MaxConnPoolSize <= 0 {
		t.Error("DefaultConfig should have MaxConnPoolSize > 0")
	}

	if config.MinConnPoolSize <= 0 {
		t.Error("DefaultConfig should have MinConnPoolSize > 0")
	}
}

func TestConfig_WithMethods(t *testing.T) {
	config := nebula.DefaultConfig().
		WithSpace("my_space").
		WithUsername("my_user").
		WithPassword("my_pass").
		WithAddresses([]string{"host1:9669", "host2:9669"}).
		WithPoolSize(5, 50)

	if config.Space != "my_space" {
		t.Errorf("Expected space my_space, got %s", config.Space)
	}

	if config.Username != "my_user" {
		t.Errorf("Expected username my_user, got %s", config.Username)
	}

	if config.Password != "my_pass" {
		t.Errorf("Expected password my_pass, got %s", config.Password)
	}

	if len(config.Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(config.Addresses))
	}

	if config.MinConnPoolSize != 5 {
		t.Errorf("Expected MinConnPoolSize 5, got %d", config.MinConnPoolSize)
	}

	if config.MaxConnPoolSize != 50 {
		t.Errorf("Expected MaxConnPoolSize 50, got %d", config.MaxConnPoolSize)
	}
}
