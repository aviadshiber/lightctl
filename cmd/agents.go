package cmd

import (
	"fmt"
	"strings"

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
			page := 0
			for {
				resp, err := appCtx.client.ListAgents(100, page)
				if err != nil {
					return fmt.Errorf("listing agents: %w", err)
				}
				agents = append(agents, resp.Data...)
				if len(resp.Data) == 0 {
					break
				}
				page++
			}
		} else {
			resp, err := appCtx.client.ListAgents(limit, 0)
			if err != nil {
				return fmt.Errorf("listing agents: %w", err)
			}
			agents = resp.Data
		}

		return printResult(appCtx, agents,
			[]string{"ID", "NAME", "HOST", "STATUS", "TAGS"},
			func() [][]string {
				rows := make([][]string, len(agents))
				for i, a := range agents {
					tags := make([]string, len(a.Tags))
					for j, t := range a.Tags {
						tags[j] = t.Name
					}
					rows[i] = []string{a.ID, a.DisplayName, a.Host, a.Status, strings.Join(tags, ",")}
				}
				return rows
			},
		)
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd)
}
