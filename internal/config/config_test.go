package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mockKeyring is an in-memory keyring for tests.
type mockKeyring struct {
	store map[string]string
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{store: make(map[string]string)}
}

func (m *mockKeyring) Get(service, user string) (string, error) {
	key := service + "/" + user
	v, ok := m.store[key]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}

func (m *mockKeyring) Set(service, user, password string) error {
	m.store[service+"/"+user] = password
	return nil
}

func (m *mockKeyring) Delete(service, user string) error {
	delete(m.store, service+"/"+user)
	return nil
}

func TestMaskAPIKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"A", "A****"},
		{"AB", "AB****"},
		{"ABCDEF", "AB****"},
		{"LRmysecrettoken123", "LR****"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := MaskAPIKey(tt.input)
			if got != tt.want {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		key   string
		valid bool
	}{
		{"api_key", true},
		{"server", true},
		{"API_KEY", true},
		{"SERVER", true},
		{"unknown", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if tt.valid && err != nil {
				t.Errorf("ValidateKey(%q) returned error: %v", tt.key, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("ValidateKey(%q) should have returned error", tt.key)
			}
		})
	}
}

func TestConfigKeyringRoundTrip(t *testing.T) {
	t.Parallel()
	kr := newMockKeyring()
	cfg := NewConfig(kr, false)
	dir := t.TempDir()
	cfg.SetConfigDir(dir)

	// Initially empty
	got, err := cfg.GetAPIKey()
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty key, got %q", got)
	}

	// Set
	if err := cfg.SetAPIKey("supersecret"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	// Get
	got, err = cfg.GetAPIKey()
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "supersecret" {
		t.Fatalf("expected supersecret, got %q", got)
	}
}

func TestConfigPlaintextRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(nil, true)
	dir := t.TempDir()
	cfg.SetConfigDir(dir)

	if err := cfg.SetAPIKey("plainkey"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	// Verify file permissions
	path := filepath.Join(dir, "config.yaml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Fatalf("expected 0600 permissions, got %04o", perm)
	}

	// Reload
	cfg2 := NewConfig(nil, true)
	cfg2.SetConfigDir(dir)
	if err := cfg2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := cfg2.GetAPIKey()
	if err != nil {
		t.Fatalf("GetAPIKey: %v", err)
	}
	if got != "plainkey" {
		t.Fatalf("expected plainkey, got %q", got)
	}
}

func TestStateFileRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(nil, true)
	cfg.SetConfigDir(t.TempDir())

	// Empty initially
	ids, err := cfg.ReadStateFile()
	if err != nil {
		t.Fatalf("ReadStateFile: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty, got %v", ids)
	}

	// Add actions
	if err := cfg.AddActionToState("action-1"); err != nil {
		t.Fatalf("AddActionToState: %v", err)
	}
	if err := cfg.AddActionToState("action-2"); err != nil {
		t.Fatalf("AddActionToState: %v", err)
	}

	ids, _ = cfg.ReadStateFile()
	if len(ids) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(ids))
	}

	// Remove one
	if err := cfg.RemoveActionFromState("action-1"); err != nil {
		t.Fatalf("RemoveActionFromState: %v", err)
	}
	ids, _ = cfg.ReadStateFile()
	if len(ids) != 1 || ids[0] != "action-2" {
		t.Fatalf("expected [action-2], got %v", ids)
	}
}

func TestAuditLog(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := AppendAuditLog(dir, "snapshot.add", "agent1", "act1", "Foo.java:42"); err != nil {
		t.Fatalf("AppendAuditLog: %v", err)
	}
	if err := AppendAuditLog(dir, "snapshot.delete", "agent1", "act1", ""); err != nil {
		t.Fatalf("AppendAuditLog: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Fatal("audit log is empty")
	}
}

func TestServerDefault(t *testing.T) {
	t.Parallel()
	cfg := NewConfig(nil, true)
	if cfg.Server != DefaultServer {
		t.Fatalf("expected %q, got %q", DefaultServer, cfg.Server)
	}
}
