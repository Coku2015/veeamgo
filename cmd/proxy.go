package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func proxyCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "proxy",
		Short: "Backup proxy inventory",
	}
	root.AddCommand(proxyListCmd())
	root.AddCommand(proxyDescribeCmd())
	return root
}

func proxyListCmd() *cobra.Command {
	var (
		nameFilter  string
		typeFilter  string
		hostID      string
		orderColumn string
		descending  bool
		limit       int
	)

	cmd := &cobra.Command{
		Use:     cmdListUse,
		Aliases: []string{"get"},
		Short:   "List backup proxies",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			filter := client.ProxyFilter{
				Name:        nameFilter,
				Type:        typeFilter,
				HostID:      hostID,
				OrderColumn: orderColumn,
				MaxItems:    limit,
			}
			if orderColumn != "" {
				asc := !descending
				filter.OrderAscending = &asc
			}

			result, err := httpClient.Proxies(ctx, filter)
			if err != nil {
				return err
			}

			states, err := httpClient.ProxyStates(ctx, filter)
			if err != nil {
				return err
			}

			stateByID := make(map[string]client.ProxyState, len(states.States))
			for _, st := range states.States {
				stateByID[st.ID] = st
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, result.Raw)
			}

			rows := make([]proxyRow, 0, len(result.Proxies))
			for _, proxy := range result.Proxies {
				status := stateByID[proxy.ID]
				rows = append(rows, proxyRow{
					Name:      proxy.Name,
					Type:      proxyTypeDisplay(proxy.Type),
					HostName:  proxy.HostName,
					MaxTasks:  proxy.MaxTaskCount,
					Online:    yesNo(status.IsOnline),
					Disabled:  yesNo(status.IsDisabled),
					OutOfDate: yesNo(status.IsOutOfDate),
				})
			}

			return output.Print(format, rows)
		},
	}

	cmd.Flags().StringVar(&nameFilter, "name", "", "Filter by proxy name (supports * wildcards)")
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by proxy type (e.g. ViProxy, HvProxy)")
	cmd.Flags().StringVar(&hostID, "host-id", "", "Filter by host identifier")
	cmd.Flags().StringVar(&orderColumn, "sort", "", "Sort results by column (e.g. name)")
	cmd.Flags().BoolVar(&descending, "desc", false, "Sort results in descending order")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of records to return (default: all)")

	return cmd
}

func proxyDescribeCmd() *cobra.Command {
	var name string
	var typeFilter string

	cmd := &cobra.Command{
		Use:     cmdDescribeUse,
		Aliases: []string{"show"},
		Short:   "Describe a backup proxy",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("specify --name <proxy>")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), time.Minute)
			defer cancel()

			httpClient, _, err := newAPIClient(ctx)
			if err != nil {
				return err
			}

			states, err := httpClient.ProxyStates(ctx, client.ProxyFilter{
				Name: name,
				Type: typeFilter,
			})
			if err != nil {
				return err
			}
			var match *client.ProxyState
			for _, st := range states.States {
				if strings.EqualFold(st.Name, name) {
					if match != nil {
						return fmt.Errorf("multiple proxies found matching %q; use --type or rename proxies", name)
					}
					state := st
					match = &state
				}
			}
			if match == nil {
				return fmt.Errorf("proxy %q not found", name)
			}

			detail, err := httpClient.ProxyDetail(ctx, match.ID)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == "json" {
				return output.Print(format, detail.Raw)
			}

			view := proxyDetailView{
				ID:                    detail.Proxy.ID,
				Name:                  detail.Proxy.Name,
				Type:                  proxyTypeDisplay(detail.Proxy.Type),
				APISourceType:         detail.Proxy.Type,
				Description:           detail.Proxy.Description,
				HostID:                detail.Proxy.HostID,
				HostName:              detail.Proxy.HostName,
				MaxTasks:              detail.Proxy.MaxTaskCount,
				TransportMode:         detail.Proxy.TransportMode,
				FailoverToNetwork:     yesNo(detail.Proxy.FailoverToNetwork),
				HostToProxyEncryption: yesNo(detail.Proxy.HostToProxyEncryption),
				AutoSelectDatastores:  yesNo(detail.Proxy.AutoSelectDatastores),
				ConnectedDatastores:   strings.Join(detail.Proxy.ConnectedDatastores, ", "),
			}

			return output.Print(format, view)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Proxy name to describe")
	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by proxy type when resolving name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

type proxyRow struct {
	Name      string `json:"Name"`
	Type      string `json:"Type"`
	HostName  string `json:"Host"`
	MaxTasks  int    `json:"Max Tasks"`
	Online    string `json:"Online"`
	Disabled  string `json:"Disabled"`
	OutOfDate string `json:"Out Of Date"`
}

type proxyDetailView struct {
	ID                    string `json:"ID"`
	Name                  string `json:"Name"`
	Type                  string `json:"Type"`
	APISourceType         string `json:"API Type"`
	Description           string `json:"Description,omitempty"`
	HostID                string `json:"Host ID,omitempty"`
	HostName              string `json:"Host,omitempty"`
	MaxTasks              int    `json:"Max Tasks,omitempty"`
	TransportMode         string `json:"Transport Mode,omitempty"`
	FailoverToNetwork     string `json:"Failover To Network"`
	HostToProxyEncryption string `json:"Host To Proxy Encryption"`
	AutoSelectDatastores  string `json:"Auto Select Datastores"`
	ConnectedDatastores   string `json:"Connected Datastores,omitempty"`
}

func proxyTypeDisplay(raw string) string {
	if raw == "" {
		return ""
	}

	switch raw {
	case "ViProxy":
		return "VMware Proxy"
	case "HvProxy":
		return "Hyper-V Proxy"
	case "HvOffhostProxy":
		return "Hyper-V Offhost Proxy"
	case "GeneralPurposeProxy":
		return "General Proxy"
	case "LinuxProxy":
		return "Linux Proxy"
	case "WindowsProxy":
		return "Windows Proxy"
	case "ProxyGateway":
		return "Gateway Proxy"
	case "ProxyAppliance":
		return "Appliance Proxy"
	case "AwsProxy":
		return "AWS Proxy"
	case "AzureProxy":
		return "Azure Proxy"
	case "ObjectStorageProxy":
		return "Object Storage Proxy"
	case "CloudProxy":
		return "Cloud Proxy"
	case "FileProxy":
		return "File Proxy"
	}

	runes := []rune(raw)
	var builder strings.Builder
	for i, r := range runes {
		if i > 0 && proxyShouldInsertSpace(runes[i-1], r) {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
	}

	return strings.TrimSpace(builder.String())
}

func proxyShouldInsertSpace(prev, current rune) bool {
	if unicode.IsLower(prev) && unicode.IsUpper(current) {
		return true
	}
	if unicode.IsLetter(prev) && unicode.IsDigit(current) {
		return true
	}
	if unicode.IsDigit(prev) && unicode.IsLetter(current) {
		return true
	}
	return false
}
