package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/veeamgo/veeamgo/internal/client"
	"github.com/veeamgo/veeamgo/pkg/output"
)

var (
	localRepositoryTypeAliases = map[string]string{
		"winlocal":       "WinLocal",
		"windows":        "WinLocal",
		"windowslocal":   "WinLocal",
		"linuxlocal":     "LinuxLocal",
		"linux":          "LinuxLocal",
		"linux_local":    "LinuxLocal",
		"linuxhardened":  "LinuxHardened",
		"hardened":       "LinuxHardened",
		"linux-hardened": "LinuxHardened",
		"smb":            "Smb",
		"cifs":           "Smb",
		"nfs":            "Nfs",
	}
	localRepositoryTypesList = []string{"WinLocal", "LinuxLocal", "LinuxHardened", "Smb", "Nfs"}
	localRepositoryTypesSet  = make(map[string]struct{})

	objectRepositoryTypeAliases = map[string]string{
		"azureblob":           "AzureBlob",
		"azure":               "AzureBlob",
		"azurearchive":        "AzureArchive",
		"azuredatabox":        "AzureDataBox",
		"amazons3":            "AmazonS3",
		"s3":                  "AmazonS3",
		"amazons3glacier":     "AmazonS3Glacier",
		"s3glacier":           "AmazonS3Glacier",
		"s3compatible":        "S3Compatible",
		"compatible":          "S3Compatible",
		"wasabicloud":         "WasabiCloud",
		"wasabi":              "WasabiCloud",
		"googlecloud":         "GoogleCloud",
		"gcp":                 "GoogleCloud",
		"ibmcloud":            "IBMCloud",
		"veeamdatacloudvault": "VeeamDataCloudVault",
	}
	objectRepositoryTypesList = []string{
		"AzureBlob",
		"AzureArchive",
		"AzureDataBox",
		"AmazonS3",
		"AmazonS3Glacier",
		"S3Compatible",
		"WasabiCloud",
		"GoogleCloud",
		"IBMCloud",
		"VeeamDataCloudVault",
	}
	objectRepositoryTypesSet = make(map[string]struct{})
)

func init() {
	for _, t := range localRepositoryTypesList {
		localRepositoryTypesSet[t] = struct{}{}
	}
	for _, t := range objectRepositoryTypesList {
		objectRepositoryTypesSet[t] = struct{}{}
	}
}

func cleanTypeKey(raw string) string {
	clean := strings.ToLower(strings.TrimSpace(raw))
	clean = strings.ReplaceAll(clean, "-", "")
	clean = strings.ReplaceAll(clean, "_", "")
	return clean
}

type repositoryTarget struct {
	ID       string
	Name     string
	Type     string
	HostName string
	State    *client.RepositoryState
	Detail   map[string]any
}

func resolveLocalRepositoryTarget(ctx context.Context, api *client.Client, name, id, repoTypeHint string) (*repositoryTarget, error) {
	return resolveRepositoryTargetWithNormalizer(ctx, api, name, id, repoTypeHint, normalizeLocalRepositoryType, isLocalRepositoryType, "local")
}

func resolveObjectRepositoryTarget(ctx context.Context, api *client.Client, name, id, repoTypeHint string) (*repositoryTarget, error) {
	return resolveRepositoryTargetWithNormalizer(ctx, api, name, id, repoTypeHint, normalizeObjectRepositoryType, isObjectRepositoryType, "object")
}

func resolveRepositoryTargetWithNormalizer(
	ctx context.Context,
	api *client.Client,
	name, id, repoTypeHint string,
	normalize func(string) (string, error),
	allowed func(string) bool,
	category string,
) (*repositoryTarget, error) {
	cleanID := strings.TrimSpace(id)
	var state *client.RepositoryState
	var err error

	if cleanID != "" {
		states, err := api.RepositoryStates(ctx, client.RepositoryStatesFilter{
			ID:       cleanID,
			MaxItems: 1,
		})
		if err != nil {
			return nil, err
		}
		if len(states) == 0 {
			return nil, fmt.Errorf("repository with id %q not found", cleanID)
		}
		state = &states[0]
	} else {
		cleanName := strings.TrimSpace(name)
		if cleanName == "" {
			return nil, fmt.Errorf("repository name cannot be empty")
		}
		state, err = api.RepositoryStateByName(ctx, cleanName)
		if err != nil {
			return nil, err
		}
		cleanID = state.ID
	}

	detail, err := api.Repository(ctx, cleanID)
	if err != nil {
		return nil, err
	}

	target := &repositoryTarget{
		ID:       cleanID,
		Name:     state.Name,
		Type:     state.Type,
		HostName: state.HostName,
		State:    state,
		Detail:   detail,
	}

	if target.Name == "" {
		if v, ok := detail["name"].(string); ok {
			target.Name = v
		}
	}
	if target.Type == "" {
		if v, ok := detail["type"].(string); ok {
			target.Type = v
		}
	}
	if target.HostName == "" {
		if v, ok := detail["hostName"].(string); ok {
			target.HostName = v
		}
	}

	if repoTypeHint != "" {
		if normalize != nil {
			expected, err := normalize(repoTypeHint)
			if err != nil {
				return nil, err
			}
			if !strings.EqualFold(target.Type, expected) {
				return nil, fmt.Errorf("repository %q is of type %s, not %s", target.Name, target.Type, expected)
			}
		}
	}

	if allowed != nil && !allowed(target.Type) {
		switch category {
		case "local":
			return nil, fmt.Errorf("repository %q is an object storage repository; use veeamgo <verb> objectrepository", target.Name)
		case "object":
			return nil, fmt.Errorf("repository %q is not an object storage repository; use veeamgo <verb> repository", target.Name)
		default:
			return nil, fmt.Errorf("repository %q is not supported by this command", target.Name)
		}
	}

	return target, nil
}

func normalizeLocalRepositoryType(raw string) (string, error) {
	clean := cleanTypeKey(raw)
	if clean == "" {
		return "", fmt.Errorf("repository type cannot be empty")
	}
	if canonical, ok := localRepositoryTypeAliases[clean]; ok {
		return canonical, nil
	}
	trimmed := strings.TrimSpace(raw)
	if _, ok := localRepositoryTypesSet[trimmed]; ok {
		return trimmed, nil
	}
	return "", fmt.Errorf("repository type %q is not supported by this command", raw)
}

func normalizeObjectRepositoryType(raw string) (string, error) {
	clean := cleanTypeKey(raw)
	if clean == "" {
		return "", fmt.Errorf("repository type cannot be empty")
	}
	if canonical, ok := objectRepositoryTypeAliases[clean]; ok {
		return canonical, nil
	}
	trimmed := strings.TrimSpace(raw)
	if _, ok := objectRepositoryTypesSet[trimmed]; ok {
		return trimmed, nil
	}
	return "", fmt.Errorf("object repository type %q is not supported", raw)
}

func localRepositoryTypes() []string {
	return append([]string(nil), localRepositoryTypesList...)
}

func objectRepositoryTypes() []string {
	return append([]string(nil), objectRepositoryTypesList...)
}

func isLocalRepositoryType(value string) bool {
	_, ok := localRepositoryTypesSet[strings.TrimSpace(value)]
	return ok
}

func isObjectRepositoryType(value string) bool {
	_, ok := objectRepositoryTypesSet[strings.TrimSpace(value)]
	return ok
}

func defaultMountCachePath(serverType string) string {
	switch serverType {
	case "WindowsHost", "WindowsServer", "WinServer":
		return `C:\ProgramData\Veeam\Backup\IRCache\`
	case "LinuxHost", "LinuxServer":
		return "/var/tmp/veeam/IRCache"
	default:
		return `C:\ProgramData\Veeam\Backup\IRCache\`
	}
}

func printRepositoryProvisioningResult(cmd *cobra.Command, session *client.Session, repoType, repoName, location string, waited bool) error {
	label := repositoryTypeDisplay(repoType)
	if label == "" {
		label = "Repository"
	}

	format := outputFormat()
	messageLocation := strings.TrimSpace(location)

	if format == "json" {
		status := "provisioning started"
		switch {
		case session == nil:
			status = "provisioning completed"
		case waited:
			status = "provisioning finished"
		}
		payload := map[string]any{
			"message": fmt.Sprintf("%s %q %s %s", label, repoName, status, messageLocation),
			"repository": map[string]string{
				"name": repoName,
				"type": repoType,
			},
			"waited": waited,
		}
		if messageLocation != "" {
			payload["location"] = messageLocation
		}
		if session != nil {
			payload["session"] = session
		}
		return output.Print(format, payload)
	}

	prefix := fmt.Sprintf("%s %q", label, repoName)
	if messageLocation != "" {
		prefix = fmt.Sprintf("%s %s", prefix, messageLocation)
	}

	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s provisioning completed.\n", prefix)
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s provisioning finished with result %s (session %s).\n", prefix, result, session.ID)
		if session.ResourceID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repository ID: %s\n", session.ResourceID)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started provisioning %s (session %s).\n", prefix, session.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}

func printRepositoryUpdateResult(cmd *cobra.Command, session *client.Session, repoType, repoName, location string, waited bool) error {
	label := repositoryTypeDisplay(repoType)
	if label == "" {
		label = "Repository"
	}

	format := outputFormat()
	messageLocation := strings.TrimSpace(location)

	if format == "json" {
		status := "update started"
		switch {
		case session == nil:
			status = "update completed"
		case waited:
			status = "update finished"
		}
		payload := map[string]any{
			"message": fmt.Sprintf("%s %q %s %s", label, repoName, status, messageLocation),
			"repository": map[string]string{
				"name": repoName,
				"type": repoType,
			},
			"waited": waited,
		}
		if messageLocation != "" {
			payload["location"] = messageLocation
		}
		if session != nil {
			payload["session"] = session
		}
		return output.Print(format, payload)
	}

	prefix := fmt.Sprintf("%s %q", label, repoName)
	if messageLocation != "" {
		prefix = fmt.Sprintf("%s %s", prefix, messageLocation)
	}

	if session == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s update completed.\n", prefix)
		return nil
	}

	if waited {
		result := session.State
		if session.Result != nil && session.Result.Result != "" {
			result = session.Result.Result
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s update finished with result %s (session %s).\n", prefix, result, session.ID)
		if session.ResourceID != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repository ID: %s\n", session.ResourceID)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Started updating %s (session %s).\n", prefix, session.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Use veeamgo get session logs --id %s to track progress.\n", session.ID)
	return nil
}
