package cmd

import (
	"fmt"
	"strings"

	internalcfg "github.com/aviadshiber/lightctl/internal/config"
	"github.com/aviadshiber/lightctl/internal/output"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage lightctl configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(args[0])
		value := args[1]

		if err := internalcfg.ValidateKey(key); err != nil {
			return err
		}

		switch key {
		case "api_key":
			if err := appCtx.cfg.SetAPIKey(value); err != nil {
				return fmt.Errorf("saving API key: %w", err)
			}
			appCtx.io.Success("API key saved")
		case "server":
			appCtx.cfg.Server = value
			if err := appCtx.cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			appCtx.io.Success(fmt.Sprintf("Server set to %s", value))
		case "agent_pool_id":
			appCtx.cfg.AgentPoolID = value
			if err := appCtx.cfg.Save(); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}
			appCtx.io.Success(fmt.Sprintf("Agent pool ID set to %s", value))
		}
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := strings.ToLower(args[0])
		if err := internalcfg.ValidateKey(key); err != nil {
			return err
		}

		switch key {
		case "api_key":
			stored, err := appCtx.cfg.GetAPIKey()
			if err != nil {
				return fmt.Errorf("reading API key: %w", err)
			}
			fmt.Fprintln(appCtx.io.Out, internalcfg.MaskAPIKey(stored))
		case "server":
			fmt.Fprintln(appCtx.io.Out, appCtx.cfg.Server)
		case "agent_pool_id":
			fmt.Fprintln(appCtx.io.Out, appCtx.cfg.AgentPoolID)
		}
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		stored, _ := appCtx.cfg.GetAPIKey()
		masked := internalcfg.MaskAPIKey(stored)

		rows := [][]string{
			{"api_key", masked},
			{"server", appCtx.cfg.Server},
			{"agent_pool_id", appCtx.cfg.AgentPoolID},
		}
		return output.PrintTable(appCtx.io.Out, []string{"KEY", "VALUE"}, rows)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
}
