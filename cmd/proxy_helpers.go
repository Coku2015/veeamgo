package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/veeamgo/veeamgo/internal/client"
)

func proxyStateByName(ctx context.Context, api *client.Client, name, proxyType string) (*client.ProxyState, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("proxy name cannot be empty")
	}

	filter := client.ProxyFilter{
		Name: name,
	}
	if trimmedType := strings.TrimSpace(proxyType); trimmedType != "" {
		filter.Type = trimmedType
	}

	states, err := api.ProxyStates(ctx, filter)
	if err != nil {
		return nil, err
	}

	var match *client.ProxyState
	for _, st := range states.States {
		if strings.EqualFold(st.Name, name) {
			if match != nil {
				return nil, fmt.Errorf("multiple proxies found matching %q; use --type or rename proxies", name)
			}
			state := st
			match = &state
		}
	}

	if match == nil {
		return nil, fmt.Errorf("proxy %q not found", name)
	}

	if strings.TrimSpace(match.ID) == "" {
		return nil, fmt.Errorf("proxy %q does not expose an identifier", name)
	}

	return match, nil
}

type proxyTarget struct {
	ID       string
	Name     string
	Type     string
	HostName string
	Disabled bool
	Detail   *client.ProxyDetail
}

func resolveProxyTarget(ctx context.Context, api *client.Client, name, id, proxyType string) (*proxyTarget, error) {
	if trimmedID := strings.TrimSpace(id); trimmedID != "" {
		detail, err := api.ProxyDetail(ctx, trimmedID)
		if err != nil {
			return nil, err
		}

		state, err := proxyStateByName(ctx, api, detail.Proxy.Name, detail.Proxy.Type)
		if err != nil {
			return nil, err
		}

		identifier := firstNonEmpty(detail.Proxy.ID, trimmedID)
		hostName := firstNonEmpty(detail.Proxy.HostName, state.HostName)

		return &proxyTarget{
			ID:       identifier,
			Name:     detail.Proxy.Name,
			Type:     detail.Proxy.Type,
			HostName: hostName,
			Disabled: state.IsDisabled,
			Detail:   detail,
		}, nil
	}

	state, err := proxyStateByName(ctx, api, name, proxyType)
	if err != nil {
		return nil, err
	}

	detail, err := api.ProxyDetail(ctx, state.ID)
	if err != nil {
		return nil, err
	}

	identifier := firstNonEmpty(detail.Proxy.ID, state.ID)
	hostName := firstNonEmpty(detail.Proxy.HostName, state.HostName)

	return &proxyTarget{
		ID:       identifier,
		Name:     detail.Proxy.Name,
		Type:     detail.Proxy.Type,
		HostName: hostName,
		Disabled: state.IsDisabled,
		Detail:   detail,
	}, nil
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
