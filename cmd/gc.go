package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Garbage-collect orphaned actions from the state file",
	Long: `Reads ~/.config/lightctl/active-actions.json and attempts to DELETE
each action from the server. Successfully deleted actions are removed
from the state file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, err := appCtx.cfg.ReadStateFile()
		if err != nil {
			return fmt.Errorf("reading state file: %w", err)
		}

		if len(ids) == 0 {
			appCtx.io.Success("No active actions to clean up")
			return nil
		}

		var deleted, failed int
		var remaining []string

		for _, id := range ids {
			if err := appCtx.client.DeleteAction(id); err != nil {
				appCtx.io.Warning(fmt.Sprintf("failed to delete %s: %v", id, err))
				remaining = append(remaining, id)
				failed++
			} else {
				appCtx.io.Success(fmt.Sprintf("Deleted %s", id))
				deleted++
			}
		}

		if err := appCtx.cfg.WriteStateFile(remaining); err != nil {
			return fmt.Errorf("updating state file: %w", err)
		}

		appCtx.io.Printf("GC complete: deleted %d, failed %d\n", deleted, failed)
		return nil
	},
}
