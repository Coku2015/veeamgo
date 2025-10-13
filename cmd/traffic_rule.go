package cmd

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/pkg/output"
)

func trafficRuleCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "trafficrule",
		Short: "Traffic throttling and preferred network rules",
	}
	root.AddCommand(newTrafficRuleGetCmd(cmdGetUse, false, nil))
	return root
}

func newTrafficRuleGetCmd(use string, hidden bool, aliases []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     use,
		Aliases: aliases,
		Short:   "List traffic rules and preferred networks",
		Hidden:  hidden,
		Args:    cobra.NoArgs,
		RunE:    runTrafficRuleGet,
	}
	return cmd
}

func runTrafficRuleGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
	defer cancel()

	httpClient, _, err := newAPIClient(ctx)
	if err != nil {
		return err
	}

	result, err := httpClient.TrafficRules(ctx)
	if err != nil {
		return err
	}

	format := outputFormat()
	if format == "json" {
		return output.Print(format, result.Raw)
	}

	rows := make([]trafficRuleRow, 0, len(result.Rules))
	for _, rule := range result.Rules {
		rows = append(rows, trafficRuleRow{
			Name:        rule.Name,
			SourceRange: ipRange(rule.SourceIPStart, rule.SourceIPEnd),
			TargetRange: ipRange(rule.TargetIPStart, rule.TargetIPEnd),
			Encryption:  yesNo(rule.EncryptionEnabled),
			Throttling:  yesNo(rule.ThrottlingEnabled),
			LimitValue:  rule.ThrottlingValue,
			LimitUnit:   rule.ThrottlingUnit,
			TimeWindow:  yesNo(rule.ThrottlingWindowEnabled),
		})
	}

	return output.Print(format, rows)
}

type trafficRuleRow struct {
	Name        string `json:"Name"`
	SourceRange string `json:"Source Range"`
	TargetRange string `json:"Target Range"`
	Encryption  string `json:"Encryption"`
	Throttling  string `json:"Throttling"`
	LimitValue  int    `json:"Limit Value"`
	LimitUnit   string `json:"Limit Unit"`
	TimeWindow  string `json:"Time Window"`
}

func ipRange(start, end string) string {
	switch {
	case start == "" && end == "":
		return ""
	case start == end || end == "":
		return start
	default:
		return strings.Join([]string{start, end}, " - ")
	}
}
