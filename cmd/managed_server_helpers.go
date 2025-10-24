package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Coku2015/veeamgo/internal/client"
)

func resolveManagedServerByName(ctx context.Context, api *client.Client, name string) (*client.ManagedServer, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("managed server name cannot be empty")
	}

	servers, err := api.ManagedServers(ctx, client.ManagedServersFilter{
		Name: clean,
	})
	if err != nil {
		return nil, err
	}

	var match *client.ManagedServer
	for _, server := range servers {
		if strings.EqualFold(server.Name, clean) {
			s := server
			if match != nil {
				return nil, fmt.Errorf("multiple managed servers matched %q; rename duplicates or use unique names", clean)
			}
			match = &s
		}
	}

	if match == nil {
		return nil, fmt.Errorf("host %q is not a managed server. Add it first using 'veeamgo add managedserver windows' or 'veeamgo add managedserver linux'", clean)
	}

	return match, nil
}

func resolveManagedServer(ctx context.Context, api *client.Client, reference string) (*client.ManagedServer, error) {
	clean := strings.TrimSpace(reference)
	if clean == "" {
		return nil, fmt.Errorf("managed server reference cannot be empty")
	}

	if looksLikeUUID(clean) {
		server, err := api.ManagedServer(ctx, clean)
		if err == nil {
			return server, nil
		}
	}

	return resolveManagedServerByName(ctx, api, clean)
}

func looksLikeUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for _, idx := range []int{8, 13, 18, 23} {
		if value[idx] != '-' {
			return false
		}
	}
	return true
}
