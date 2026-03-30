package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultServer is the default LightRun API server.
	DefaultServer = "https://app.lightrun.com"

	// KeyringService is the service name used in the OS keychain.
	KeyringService = "lightctl"
	// KeyringUser is the keychain account for the API key.
	KeyringUser = "api_key"

	stateFileName = "active-actions.json"
	configDirName = "lightctl"
	configFile    = "config.yaml"
	auditFile     = "audit.log"
)

// Keyring abstracts OS keychain operations so it can be mocked in tests.
type Keyring interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

// Config holds application configuration.
type Config struct {
	Server string `yaml:"server,omitempty"`
	APIKey string `yaml:"api_key,omitempty"` // only used in plaintext mode

	keyring              Keyring
	insecurePlaintextCfg bool
	configDir            string
	mu                   sync.Mutex
}

// NewConfig creates a Config. When insecurePlaintext is false the api_key is
// stored in the OS keychain via kr. When true it is stored in the YAML file.
func NewConfig(kr Keyring, insecurePlaintext bool) *Config {
	return &Config{
		Server:               DefaultServer,
		keyring:              kr,
		insecurePlaintextCfg: insecurePlaintext,
	}
}

// ConfigDir returns the configuration directory, creating it if needed.
func (c *Config) ConfigDir() (string, error) {
	if c.configDir != "" {
		return c.configDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", configDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	c.configDir = dir
	return dir, nil
}

// SetConfigDir overrides the config directory (useful for tests).
func (c *Config) SetConfigDir(dir string) {
	c.configDir = dir
}

// Load reads the YAML config file if it exists.
func (c *Config) Load() error {
	dir, err := c.ConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	return nil
}

// Save writes the YAML config file with 0600 permissions.
func (c *Config) Save() error {
	dir, err := c.ConfigDir()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	path := filepath.Join(dir, configFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// GetAPIKey resolves the API key using the precedence: keyring/file > default.
// The caller is responsible for checking flag/env overrides before calling.
func (c *Config) GetAPIKey() (string, error) {
	if c.insecurePlaintextCfg {
		return c.APIKey, nil
	}
	if c.keyring == nil {
		return "", fmt.Errorf("keyring not available")
	}
	key, err := c.keyring.Get(KeyringService, KeyringUser)
	if err != nil {
		// Keyring miss is not necessarily a hard error – the user may not
		// have configured one yet.
		return "", nil
	}
	return key, nil
}

// SetAPIKey persists the API key.
func (c *Config) SetAPIKey(key string) error {
	if c.insecurePlaintextCfg {
		c.APIKey = key
		return c.Save()
	}
	if c.keyring == nil {
		return fmt.Errorf("keyring not available; use --insecure-plaintext-config to store in plaintext")
	}
	return c.keyring.Set(KeyringService, KeyringUser, key)
}

// MaskAPIKey returns a masked version of the key (e.g. "LR****").
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) == 1 {
		return key[:1] + "****"
	}
	return key[:2] + "****"
}

// --- State file (active-actions.json) ---

// ReadStateFile reads action IDs from the state file.
func (c *Config) ReadStateFile() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readStateFileLocked()
}

// WriteStateFile writes action IDs to the state file.
// Note: the mutex guards within-process races only; cross-process races
// (multiple concurrent lightctl watch invocations) require OS-level locking.
func (c *Config) WriteStateFile(ids []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeStateFileLocked(ids)
}

// AddActionToState appends an action ID to the state file atomically.
func (c *Config) AddActionToState(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids, err := c.readStateFileLocked()
	if err != nil {
		return err
	}
	ids = append(ids, id)
	return c.writeStateFileLocked(ids)
}

// RemoveActionFromState removes an action ID from the state file atomically.
func (c *Config) RemoveActionFromState(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids, err := c.readStateFileLocked()
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(ids))
	for _, existing := range ids {
		if existing != id {
			filtered = append(filtered, existing)
		}
	}
	return c.writeStateFileLocked(filtered)
}

// readStateFileLocked reads the state file. Caller must hold c.mu.
func (c *Config) readStateFileLocked() ([]string, error) {
	dir, err := c.ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}
	return ids, nil
}

// writeStateFileLocked writes the state file. Caller must hold c.mu.
func (c *Config) writeStateFileLocked(ids []string) error {
	dir, err := c.ConfigDir()
	if err != nil {
		return err
	}
	if ids == nil {
		ids = []string{}
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshalling state file: %w", err)
	}
	path := filepath.Join(dir, stateFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing state file: %w", err)
	}
	return nil
}

// --- Audit log ---

// AppendAuditLog writes a structured line to the audit log.
func AppendAuditLog(cfgDir, op, agentID, actionID, fileLine string) error {
	path := filepath.Join(cfgDir, auditFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s  %s  agent=%s  action=%s  file=%s\n",
		ts, op, agentID, actionID, fileLine)
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("writing audit log: %w", err)
	}
	return nil
}

// ValidateKey checks that a config key name is supported.
func ValidateKey(key string) error {
	switch strings.ToLower(key) {
	case "api_key", "server":
		return nil
	default:
		return fmt.Errorf("unknown config key %q (valid keys: api_key, server)", key)
	}
}
