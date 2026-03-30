package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aviadshiber/lightctl/internal/client"
	"github.com/aviadshiber/lightctl/internal/config"
	"github.com/aviadshiber/lightctl/internal/iostreams"
	"github.com/aviadshiber/lightctl/internal/output"
)

// appContext bundles shared state that every command needs.
type appContext struct {
	cfg    *config.Config
	client *client.Client
	io     *iostreams.IOStreams
}

// parseFileLine splits "File.java:42" into filename and line number.
func parseFileLine(s string) (string, int, error) {
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return "", 0, fmt.Errorf("expected <file>:<line> format, got %q", s)
	}
	file := s[:idx]
	line, err := strconv.Atoi(s[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid line number in %q: %w", s, err)
	}
	if line <= 0 {
		return "", 0, fmt.Errorf("line number must be positive, got %d", line)
	}
	return file, line, nil
}

// printResult handles --output, --pretty, --jq for a given data value.
func printResult(ctx *appContext, data interface{}, headers []string, rowsFn func() [][]string) error {
	jqExpr, _ := rootCmd.Flags().GetString("jq")
	if jqExpr != "" {
		filtered, err := output.FilterJQ(data, jqExpr)
		if err != nil {
			return err
		}
		data = filtered
		// After jq, always output JSON
		pretty, _ := rootCmd.Flags().GetBool("pretty")
		if pretty {
			return output.PrintPrettyJSON(ctx.io.Out, data)
		}
		return output.PrintJSON(ctx.io.Out, data)
	}

	outputFmt, _ := rootCmd.Flags().GetString("output")
	if outputFmt == "table" && rowsFn != nil {
		return output.PrintTable(ctx.io.Out, headers, rowsFn())
	}

	pretty, _ := rootCmd.Flags().GetBool("pretty")
	if pretty {
		return output.PrintPrettyJSON(ctx.io.Out, data)
	}
	return output.PrintJSON(ctx.io.Out, data)
}
