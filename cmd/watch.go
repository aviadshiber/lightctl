package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/aviadshiber/lightctl/internal/config"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch <agent-id> <file>:<line> <expr>",
	Short: "Create a snapshot and poll until the expression is captured",
	Long: `Creates a temporary snapshot at the given location and polls until the
expression is captured or the timeout expires. On success the value is
printed and the snapshot is cleaned up. On timeout the action ID is
printed for manual cleanup (exit code 8).`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		fileName, lineNum, err := parseFileLine(args[1])
		if err != nil {
			return err
		}
		expr := args[2]
		timeout, _ := cmd.Flags().GetInt("timeout")
		interval, _ := cmd.Flags().GetInt("interval")

		// 1. Create snapshot
		action, err := appCtx.client.CreateAction(client.CreateActionRequest{
			AgentID:    agentID,
			ActionType: "SNAPSHOT",
			Location:   fileName,
			Line:       lineNum,
		})
		if err != nil {
			return fmt.Errorf("creating watch snapshot: %w", err)
		}
		appCtx.io.Success(fmt.Sprintf("Snapshot %s created, watching for %q", action.ID, expr))

		// 2. Persist to state file
		if err := appCtx.cfg.AddActionToState(action.ID); err != nil {
			appCtx.io.Warning(fmt.Sprintf("failed to persist action: %v", err))
		}

		cfgDir, _ := appCtx.cfg.ConfigDir()
		_ = config.AppendAuditLog(cfgDir, "watch.start", agentID, action.ID, args[1])

		// 3. Signal handler for cleanup
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		cleanup := func() {
			if delErr := appCtx.client.DeleteAction(action.ID); delErr != nil {
				appCtx.io.Warning(fmt.Sprintf("failed to delete action %s: %v", action.ID, delErr))
			} else {
				appCtx.io.Success(fmt.Sprintf("Cleaned up action %s", action.ID))
			}
			_ = appCtx.cfg.RemoveActionFromState(action.ID)
			_ = config.AppendAuditLog(cfgDir, "watch.cleanup", agentID, action.ID, args[1])
		}

		// 4. Poll loop
		start := time.Now()
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		timeoutDuration := time.Duration(timeout) * time.Second

		for {
			select {
			case <-sigCh:
				signal.Stop(sigCh)
				appCtx.io.Warning("Interrupted, cleaning up...")
				cleanup()
				os.Exit(130)
			case <-ticker.C:
				if time.Since(start) > timeoutDuration {
					appCtx.io.Error(fmt.Sprintf("Timeout after %ds, cleaning up action %s...",
						timeout, action.ID))
					cleanup()
					os.Exit(8)
				}

				a, err := appCtx.client.GetAction(action.ID)
				if err != nil {
					appCtx.io.Warning(fmt.Sprintf("poll error: %v", err))
					continue
				}

				// Check if snapshot fired (status changed from ACCEPTED)
				if a.Status != "ACCEPTED" {
					// Look for the expression in the snapshot data.
					// The snapshot data is returned as part of the action when fired.
					appCtx.io.Success(fmt.Sprintf("Snapshot fired (status: %s)", a.Status))
					appCtx.io.Printf("Expression: %s\n", expr)
					appCtx.io.Printf("Action details:\n")
					if err := printResult(appCtx, a, nil, nil); err != nil {
						appCtx.io.Warning(fmt.Sprintf("failed to print result: %v", err))
					}
					cleanup()
					return nil
				}
			}
		}
	},
}

func init() {
	watchCmd.Flags().Int("timeout", 60, "Timeout in seconds")
	watchCmd.Flags().Int("interval", 3, "Poll interval in seconds")
}
