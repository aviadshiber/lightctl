package cmd

import (
	"fmt"
	"strconv"

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
			ActionType:  "SNAPSHOT",
			Location:    fileName,
			Line:        lineNum,
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
			offset := 0
			for {
				resp, err := appCtx.client.ListActions(agentID, "SNAPSHOT", 100, offset)
				if err != nil {
					return fmt.Errorf("listing snapshots: %w", err)
				}
				actions = append(actions, resp.Items...)
				if !resp.HasMore {
					break
				}
				offset += len(resp.Items)
			}
		} else {
			resp, err := appCtx.client.ListActions(agentID, "SNAPSHOT", limit, 0)
			if err != nil {
				return fmt.Errorf("listing snapshots: %w", err)
			}
			actions = resp.Items
		}

		return printResult(appCtx, actions,
			[]string{"ID", "TYPE", "LOCATION", "LINE", "STATUS", "CREATED"},
			func() [][]string {
				rows := make([][]string, len(actions))
				for i, a := range actions {
					rows[i] = []string{a.ID, a.ActionType, a.Location, strconv.Itoa(a.Line), a.Status, a.CreatedAt}
				}
				return rows
			},
		)
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actionID := args[0]

		if err := appCtx.client.DeleteAction(actionID); err != nil {
			return fmt.Errorf("deleting snapshot: %w", err)
		}

		if err := appCtx.cfg.RemoveActionFromState(actionID); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to remove action from state file: %v", err))
		}

		cfgDir, _ := appCtx.cfg.ConfigDir()
		if err := config.AppendAuditLog(cfgDir, "snapshot.delete", "", actionID, ""); err != nil {
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
	snapshotCmd.AddCommand(snapshotDeleteCmd)
}
