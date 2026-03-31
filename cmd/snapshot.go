package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/aviadshiber/lightctl/internal/config"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage snapshots",
}

var snapshotAddCmd = &cobra.Command{
	Use:   "add <agent-id> <file>:<line>",
	Short: "Create a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		fileName, lineNum, err := parseFileLine(args[1])
		if err != nil {
			return err
		}

		condition, _ := cmd.Flags().GetString("condition")
		expire, _ := cmd.Flags().GetInt("expire")
		maxHits, _ := cmd.Flags().GetInt("max-hits")

		action, err := appCtx.client.CreateAction(client.CreateActionRequest{
			AgentID:     agentID,
			Type:        "SNAPSHOT",
			FileName:    fileName,
			LineNumber:  lineNum,
			Condition:   condition,
			ExpireSecs:  expire,
			MaxHitCount: maxHits,
		})
		if err != nil {
			return fmt.Errorf("creating snapshot: %w", err)
		}

		// Persist to state file
		if err := appCtx.cfg.AddActionToState(action.ID); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to persist action to state file: %v", err))
		}

		// Audit log
		cfgDir, _ := appCtx.cfg.ConfigDir()
		if err := config.AppendAuditLog(cfgDir, "snapshot.add", agentID, action.ID, args[1]); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to write audit log: %v", err))
		}

		appCtx.io.Success(fmt.Sprintf("Snapshot %s created", action.ID))
		return printResult(appCtx, action, nil, nil)
	},
}

var snapshotListCmd = &cobra.Command{
	Use:   "list <agent-id>",
	Short: "List snapshots for an agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		limit, _ := cmd.Flags().GetInt("limit")
		all, _ := cmd.Flags().GetBool("all")

		var actions []client.Action

		if all {
			page := 0
			for {
				resp, err := appCtx.client.ListActions(agentID, "SNAPSHOT", 100, page)
				if err != nil {
					return fmt.Errorf("listing snapshots: %w", err)
				}
				actions = append(actions, resp.Data...)
				if len(resp.Data) == 0 {
					break
				}
				page++
			}
		} else {
			resp, err := appCtx.client.ListActions(agentID, "SNAPSHOT", limit, 0)
			if err != nil {
				return fmt.Errorf("listing snapshots: %w", err)
			}
			actions = resp.Data
		}

		return printResult(appCtx, actions,
			[]string{"ID", "TYPE", "FILE", "LINE", "STATUS", "CREATED"},
			func() [][]string {
				rows := make([][]string, len(actions))
				for i, a := range actions {
					created := time.UnixMilli(a.CreateTime).Format(time.RFC3339)
					rows[i] = []string{a.ID, a.Type, a.FileName, strconv.Itoa(a.LineNumber), a.Status, created}
				}
				return rows
			},
		)
	},
}

var snapshotGetCmd = &cobra.Command{
	Use:   "get <snapshot-id>",
	Short: "Get snapshot details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action, err := appCtx.client.GetAction(args[0])
		if err != nil {
			return fmt.Errorf("getting snapshot: %w", err)
		}
		return printResult(appCtx, action, nil, nil)
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <agent-id> <snapshot-id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		actionID := args[1]

		if err := appCtx.client.DeleteAction(actionID); err != nil {
			return fmt.Errorf("deleting snapshot: %w", err)
		}

		if err := appCtx.cfg.RemoveActionFromState(actionID); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to remove action from state file: %v", err))
		}

		cfgDir, _ := appCtx.cfg.ConfigDir()
		if err := config.AppendAuditLog(cfgDir, "snapshot.delete", agentID, actionID, ""); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to write audit log: %v", err))
		}

		appCtx.io.Success(fmt.Sprintf("Snapshot %s deleted", actionID))
		return nil
	},
}

func init() {
	snapshotAddCmd.Flags().String("condition", "", "Conditional expression")
	snapshotAddCmd.Flags().Int("expire", 0, "Expiration in seconds")
	snapshotAddCmd.Flags().Int("max-hits", 0, "Maximum hit count")

	snapshotCmd.AddCommand(snapshotAddCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotGetCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
}
