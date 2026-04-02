package cmd

import (
	"fmt"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage LightRun agents",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		all, _ := cmd.Flags().GetBool("all")

		var agents []client.Agent

		if all {
			offset := 0
			for {
				resp, err := appCtx.client.ListAgents(100, offset)
				if err != nil {
					return fmt.Errorf("listing agents: %w", err)
				}
				agents = append(agents, resp.Items...)
				if !resp.HasMore {
					break
				}
				offset += len(resp.Items)
			}
		} else {
			resp, err := appCtx.client.ListAgents(limit, 0)
			if err != nil {
				return fmt.Errorf("listing agents: %w", err)
			}
			agents = resp.Items
		}

		return printResult(appCtx, agents,
			[]string{"ID", "NAME", "TYPE", "VERSION", "STATUS"},
			func() [][]string {
				rows := make([][]string, len(agents))
				for i, a := range agents {
					rows[i] = []string{a.ID, a.Name, a.Type, a.Version, a.VersionStatus}
				}
				return rows
			},
		)
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd)
}
