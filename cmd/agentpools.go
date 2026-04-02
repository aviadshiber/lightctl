package cmd

import (
	"fmt"
	"strconv"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/spf13/cobra"
)

var agentPoolsCmd = &cobra.Command{
	Use:   "agent-pools",
	Short: "List LightRun agent pools",
}

var agentPoolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List agent pools",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		all, _ := cmd.Flags().GetBool("all")

		var pools []client.AgentPool

		if all {
			offset := 0
			for {
				resp, err := appCtx.client.GetAgentPools(100, offset)
				if err != nil {
					return fmt.Errorf("listing agent pools: %w", err)
				}
				pools = append(pools, resp.Items...)
				if !resp.HasMore {
					break
				}
				offset += len(resp.Items)
			}
		} else {
			resp, err := appCtx.client.GetAgentPools(limit, 0)
			if err != nil {
				return fmt.Errorf("listing agent pools: %w", err)
			}
			pools = resp.Items
		}

		return printResult(appCtx, pools,
			[]string{"ID", "NAME", "LIVE_AGENTS", "ENABLED"},
			func() [][]string {
				rows := make([][]string, len(pools))
				for i, p := range pools {
					rows[i] = []string{p.ID, p.Name, strconv.Itoa(p.LiveAgentsCount), strconv.FormatBool(p.AgentsEnabled)}
				}
				return rows
			},
		)
	},
}

func init() {
	agentPoolsCmd.AddCommand(agentPoolsListCmd)
}
