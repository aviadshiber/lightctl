package cmd

import (
	"fmt"
	"os"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/aviadshiber/lightctl/internal/config"
	"github.com/aviadshiber/lightctl/internal/iostreams"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var (
	appCtx *appContext

	rootCmd = &cobra.Command{
		Use:   "lightctl",
		Short: "LightRun CLI -- runtime debugging tool",
		Long: `lightctl is a thin wrapper around the LightRun REST API for runtime
debugging of production JVMs. It supports snapshots, watches, and
agent management.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
)

// realKeyring delegates to the OS keychain via go-keyring.
type realKeyring struct{}

func (realKeyring) Get(service, user string) (string, error)       { return keyring.Get(service, user) }
func (realKeyring) Set(service, user, password string) error        { return keyring.Set(service, user, password) }
func (realKeyring) Delete(service, user string) error               { return keyring.Delete(service, user) }

func init() {
	// Persistent flags
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides keychain/config)")
	rootCmd.PersistentFlags().String("server", "", "LightRun server URL")
	rootCmd.PersistentFlags().String("agent-pool-id", "", "LightRun agent pool ID (auto-discovered if not set)")
	rootCmd.PersistentFlags().String("company-id", "", "LightRun company ID (required for snapshot add / watch)")
	rootCmd.PersistentFlags().Bool("insecure-http", false, "Allow plain HTTP (no TLS)")
	rootCmd.PersistentFlags().Bool("insecure-plaintext-config", false, "Store API key in plaintext config instead of keychain")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress informational output")
	rootCmd.PersistentFlags().Bool("pretty", false, "Pretty-print JSON output")
	rootCmd.PersistentFlags().String("output", "json", "Output format: json|table")
	rootCmd.PersistentFlags().String("jq", "", "jq expression to filter JSON output")
	rootCmd.PersistentFlags().Int("limit", 50, "Number of items per page")
	rootCmd.PersistentFlags().Bool("all", false, "Fetch all pages (override --limit)")

	// Build shared context before each command (except config & version)
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ios := iostreams.New()
		quiet, _ := cmd.Flags().GetBool("quiet")
		ios.SetQuiet(quiet)

		insecurePlaintext, _ := cmd.Flags().GetBool("insecure-plaintext-config")
		cfg := config.NewConfig(realKeyring{}, insecurePlaintext)
		if err := cfg.Load(); err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		appCtx = &appContext{cfg: cfg, io: ios}

		// Commands that don't need a client
		if isConfigOrVersionCmd(cmd) {
			return nil
		}

		// Resolve server: flag > env > config > default
		server := resolveString(cmd, "server", "LIGHTCTL_SERVER", cfg.Server, config.DefaultServer)
		// Resolve API key: flag > env > keychain/config
		storedKey, _ := cfg.GetAPIKey()
		apiKey := resolveString(cmd, "api-key", "LIGHTCTL_API_KEY", storedKey, "")

		if apiKey == "" {
			return fmt.Errorf("API key required: set via --api-key, LIGHTCTL_API_KEY, or `lightctl config set api_key <key>`")
		}

		insecureHTTP, _ := cmd.Flags().GetBool("insecure-http")
		c, err := client.New(server, apiKey, client.WithInsecureHTTP(insecureHTTP))
		if err != nil {
			return err
		}

		// Resolve agent pool ID: flag > env > config > auto-discover
		agentPoolID := resolveString(cmd, "agent-pool-id", "LIGHTCTL_AGENT_POOL_ID", cfg.AgentPoolID, "")
		if agentPoolID != "" {
			c.SetAgentPoolID(agentPoolID)
		} else if !isAgentPoolFreeCmd(cmd) {
			if err := c.AutoDiscoverPool(); err != nil {
				return fmt.Errorf("agent pool ID required: set via --agent-pool-id, LIGHTCTL_AGENT_POOL_ID, or `lightctl config set agent_pool_id <id>` (%w)", err)
			}
		}

		// Resolve company ID: flag > env > config (required for create-action commands)
		if companyID := resolveString(cmd, "company-id", "LIGHTCTL_COMPANY_ID", cfg.CompanyID, ""); companyID != "" {
			c.SetCompanyID(companyID)
		}

		appCtx.client = c
		return nil
	}

	// Register sub-commands
	rootCmd.AddCommand(agentPoolsCmd)
	rootCmd.AddCommand(agentsCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(gcCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute is the main entry point.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// isConfigOrVersionCmd returns true for commands that don't need an API client.
func isConfigOrVersionCmd(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c == configCmd || c == versionCmd {
			return true
		}
	}
	return false
}

// isAgentPoolFreeCmd returns true for commands that work without an agent pool
// (e.g. agent-pools list, which is how users discover their pool ID).
func isAgentPoolFreeCmd(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c == agentPoolsCmd {
			return true
		}
	}
	return false
}

// resolveString returns the first non-empty value from: flag, env var, stored, fallback.
func resolveString(cmd *cobra.Command, flag, envVar, stored, fallback string) string {
	if v, _ := cmd.Flags().GetString(flag); v != "" {
		return v
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	if stored != "" {
		return stored
	}
	return fallback
}
