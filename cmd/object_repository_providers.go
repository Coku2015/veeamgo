package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Coku2015/veeamgo/pkg/helptext"
	"github.com/Coku2015/veeamgo/pkg/output"
)

type objectRepositoryProvider struct {
	Canonical       string
	CommandUse      string
	DisplayName     string
	Description     string
	Aliases         []string
	newConfigurator func() objectRepositoryConfigurator
}

type objectRepositoryConfigurator interface {
	BindFlags(cmd *cobra.Command)
	BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error)
}

func objectRepositoryProviderList() []objectRepositoryProvider {
	return []objectRepositoryProvider{
		newAmazonS3Provider(),
		newS3CompatibleProvider(),
		newWasabiCloudProvider(),
		newAzureBlobProvider(),
		newAzureArchiveProvider(),
		newVeeamDataCloudVaultProvider(),
	}
}

type objectRepositoryTypeInfo struct {
	Type        string
	DisplayName string
	Description string
	HasCommand  bool
}

func objectRepositoryTypeCatalog() []objectRepositoryTypeInfo {
	infos := make([]objectRepositoryTypeInfo, 0, len(objectRepositoryTypesList))
	commandMap := make(map[string]struct{})
	for _, provider := range objectRepositoryProviderList() {
		commandMap[provider.Canonical] = struct{}{}
	}

	for _, t := range objectRepositoryTypesList {
		info := objectRepositoryTypeInfo{
			Type:        t,
			DisplayName: repositoryTypeDisplay(t),
			Description: objectRepositoryTypeDescription(t),
		}
		if _, ok := commandMap[t]; ok {
			info.HasCommand = true
		}
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].DisplayName < infos[j].DisplayName
	})

	return infos
}

func objectRepositoryTypeDescription(t string) string {
	switch t {
	case "AmazonS3":
		return "Amazon Simple Storage Service (standard/Infrequent Access)."
	case "AmazonS3Glacier":
		return "Amazon S3 Glacier with optional Deep Archive tier."
	case "S3Compatible":
		return "Generic S3-compatible object storage endpoint."
	case "WasabiCloud":
		return "Wasabi Cloud Storage."
	case "GoogleCloud":
		return "Google Cloud Storage buckets."
	case "IBMCloud":
		return "IBM Cloud Object Storage."
	case "AzureBlob":
		return "Microsoft Azure Blob storage accounts."
	case "AzureArchive":
		return "Azure Blob storage using Archive tier."
	case "AzureDataBox":
		return "Azure Data Box device import jobs."
	case "VeeamDataCloudVault":
		return "Veeam Data Cloud Vault managed storage."
	default:
		return ""
	}
}

func newObjectRepositoryProviderAddCmd(provider objectRepositoryProvider) *cobra.Command {
	opts := objectRepositoryAddOptions{}
	cfg := provider.newConfigurator()

	article := "a"
	if name := strings.TrimSpace(provider.DisplayName); name != "" {
		first := strings.ToLower(string([]rune(name)[0]))
		if strings.ContainsAny(first, "aeiou") {
			article = "an"
		}
	}

	cmd := &cobra.Command{
		Use:   provider.CommandUse,
		Short: fmt.Sprintf("Add %s %s repository", article, provider.DisplayName),
		Long:  provider.Description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			disableSet := cmd.Flags().Changed("disable")
			opts.repoType = provider.Canonical
			base, err := cfg.BuildBase(cmd, &opts)
			if err != nil {
				return err
			}
			return runObjectRepositoryAddWithBase(cmd, &opts, disableSet, base)
		},
	}
	cmd.Aliases = append(cmd.Aliases, provider.Aliases...)

	cmd.Flags().StringVar(&opts.specPath, "spec", "", "Path to JSON spec describing the repository (use '-' for stdin)")
	cmd.Flags().StringSliceVar(&opts.setPairs, "set", nil, "Override spec value (repeatable, dot notation, e.g. bucket.bucketName=my-bucket)")
	cmd.Flags().StringVar(&opts.name, "name", "", "Repository name")
	cmd.Flags().StringVar(&opts.description, "description", "", "Repository description")
	cmd.Flags().BoolVar(&opts.disable, "disable", false, "Create the repository in a disabled state")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for the provisioning session to finish")
	cmd.Flags().BoolVar(&opts.yes, "yes", false, "Confirm without prompting (required to execute)")

	cfg.BindFlags(cmd)
	return cmd
}

func objectRepositoryAddTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "types",
		Short: helptext.ObjectRepositoryProvidersShort,
		RunE: func(cmd *cobra.Command, args []string) error {
			format := outputFormat()
			catalog := objectRepositoryTypeCatalog()

			if format == "json" {
				payload := make([]map[string]any, 0, len(catalog))
				for _, info := range catalog {
					payload = append(payload, map[string]any{
						"type":        info.Type,
						"display":     info.DisplayName,
						"description": info.Description,
						"hasScaffold": info.HasCommand,
					})
				}
				return output.Print(format, payload)
			}

			type typeRow struct {
				Provider    string `json:"Provider"`
				Type        string `json:"Type"`
				Description string `json:"Description"`
				Notes       string `json:"Notes"`
			}

			rows := make([]typeRow, 0, len(catalog))
			for _, info := range catalog {
				flags := ""
				if info.HasCommand {
					flags = "template available"
				}
				rows = append(rows, typeRow{
					Provider:    info.DisplayName,
					Type:        info.Type,
					Description: info.Description,
					Notes:       flags,
				})
			}

			return output.Print(format, rows)
		},
	}

	return cmd
}

func fixedCompletion(options []string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		toLower := strings.ToLower(strings.TrimSpace(toComplete))
		matches := make([]string, 0, len(options))
		for _, opt := range options {
			if toLower == "" || strings.HasPrefix(strings.ToLower(opt), toLower) {
				matches = append(matches, opt)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

func normalizeEnum(value string, options []string, flagName string) (string, error) {
	clean := strings.TrimSpace(value)
	for _, opt := range options {
		if strings.EqualFold(clean, opt) {
			return opt, nil
		}
	}
	if flagName != "" {
		return "", fmt.Errorf("invalid value %q for --%s (expected one of %s)", value, flagName, strings.Join(options, ", "))
	}
	return "", fmt.Errorf("invalid value %q (expected one of %s)", value, strings.Join(options, ", "))
}

func trimSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func ensureString(path []string, flagName, description string, payload map[string]any) error {
	if strings.TrimSpace(nestedString(payload, path...)) == "" {
		if flagName != "" {
			return fmt.Errorf("provide %s using --%s or include %s in the spec", description, flagName, strings.Join(path, "."))
		}
		return fmt.Errorf("%s is required", description)
	}
	return nil
}

func ensureBool(path []string, flagName, description string, payload map[string]any) error {
	if _, ok := nestedBool(payload, path...); !ok {
		if flagName != "" {
			return fmt.Errorf("provide %s using --%s or include %s in the spec", description, flagName, strings.Join(path, "."))
		}
		return fmt.Errorf("%s is required", description)
	}
	return nil
}

func valueExists(payload map[string]any, path ...string) bool {
	_, ok := nestedValue(payload, path...)
	return ok
}

// ------------------- Amazon S3 -------------------

type amazonS3Configurator struct {
	credentialsID     string
	regionScope       string
	connectionType    string
	gatewayIDs        []string
	bucketName        string
	folderName        string
	bucketRegion      string
	mountServerID     string
	mountServerCache  string
	mountServerVPower bool
	immutabilityOn    bool
	immutabilityDays  int
}

var (
	s3RegionScopes      = []string{"Global", "China", "Government"}
	repoConnectionTypes = []string{"Direct", "SelectedGateway"}
)

func newAmazonS3Provider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "AmazonS3",
		CommandUse:  "amazon-s3",
		DisplayName: "Amazon S3",
		Description: "Add an Amazon S3 repository without crafting JSON by hand.",
		newConfigurator: func() objectRepositoryConfigurator {
			return &amazonS3Configurator{
				regionScope:       "Global",
				connectionType:    "Direct",
				mountServerVPower: true,
			}
		},
	}
}

func (c *amazonS3Configurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.credentialsID, "credentials-id", "", "Cloud credentials ID for Amazon S3 access")
	flags.StringVar(&c.bucketName, "bucket-name", "", "Target S3 bucket name")
	flags.StringVar(&c.folderName, "folder", "", "Folder/prefix inside the bucket")
	flags.StringVar(&c.bucketRegion, "region-id", "", "AWS region identifier (for example us-east-1)")
	flags.StringVar(&c.regionScope, "region-scope", c.regionScope, "AWS partition (Global|China|Government)")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", c.mountServerVPower, "Enable vPower NFS on the mount server")
	flags.BoolVar(&c.immutabilityOn, "immutability-enabled", false, "Enable object lock immutability for stored backups")
	flags.IntVar(&c.immutabilityDays, "immutability-days", 0, "Immutability retention period in days")

	_ = cmd.RegisterFlagCompletionFunc("region-scope", fixedCompletion(s3RegionScopes))
	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *amazonS3Configurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error

	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultAmazonS3Spec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "AmazonS3"); err != nil {
		return nil, err
	}

	if flags.Changed("credentials-id") {
		if strings.TrimSpace(c.credentialsID) == "" {
			return nil, fmt.Errorf("--credentials-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "credentialsId"}, strings.TrimSpace(c.credentialsID)); err != nil {
			return nil, err
		}
	}

	currentRegionType := nestedString(payload, "account", "regionType")
	if flags.Changed("region-scope") || strings.TrimSpace(currentRegionType) == "" {
		normalized, normErr := normalizeEnum(c.regionScope, s3RegionScopes, "region-scope")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "regionType"}, normalized); err != nil {
			return nil, err
		}
	} else if _, normErr := normalizeEnum(currentRegionType, s3RegionScopes, "account.regionType"); normErr != nil {
		return nil, normErr
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		ids := trimSlice(c.gatewayIDs)
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, ids); err != nil {
			return nil, err
		}
	}

	if flags.Changed("bucket-name") {
		if err := setNestedField(payload, []string{"bucket", "bucketName"}, strings.TrimSpace(c.bucketName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"bucket", "folderName"}, strings.TrimSpace(c.folderName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("region-id") {
		if err := setNestedField(payload, []string{"bucket", "regionId"}, strings.TrimSpace(c.bucketRegion)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("immutability-enabled") {
		if err := setNestedField(payload, []string{"bucket", "immutability", "isEnabled"}, c.immutabilityOn); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-days") {
		if c.immutabilityDays < 0 {
			return nil, fmt.Errorf("--immutability-days cannot be negative")
		}
		if err := setNestedField(payload, []string{"bucket", "immutability", "daysCount"}, c.immutabilityDays); err != nil {
			return nil, err
		}
	}

	mountType := strings.TrimSpace(nestedString(payload, "mountServer", "mountServerSettingsType"))
	if mountType == "" || mountType == "windows" {
		if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
			return nil, err
		}
		if flags.Changed("mount-server-id") {
			if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
				return nil, err
			}
		}
		if flags.Changed("mount-server-cache") {
			if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
				return nil, err
			}
		}
		if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
			if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
				return nil, err
			}
		}
	} else {
		// Respect the mount server settings provided by the spec for non-windows types.
	}

	if err := ensureString([]string{"account", "credentialsId"}, "credentials-id", "cloud credential ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "bucketName"}, "bucket-name", "bucket name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "folderName"}, "folder", "folder name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "regionId"}, "region-id", "bucket region", payload); err != nil {
		return nil, err
	}

	if connectionType == "SelectedGateway" {
		ids := nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")
		if len(ids) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}

	mountType = strings.TrimSpace(nestedString(payload, "mountServer", "mountServerSettingsType"))
	if mountType == "" || mountType == "windows" {
		if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
			return nil, err
		}
		if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
			return nil, err
		}
		if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func defaultAmazonS3Spec() map[string]any {
	return map[string]any{
		"type":        "AmazonS3",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"regionType": "Global",
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"bucket": map[string]any{
			"bucketName": "",
			"folderName": "",
			"regionId":   "",
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}

// ------------------- S3 Compatible -------------------

type s3CompatibleConfigurator struct {
	credentialsID     string
	servicePoint      string
	regionID          string
	connectionType    string
	gatewayIDs        []string
	bucketName        string
	folderName        string
	mountServerID     string
	mountServerCache  string
	mountServerVPower bool
	immutabilityOn    bool
	immutabilityDays  int
}

func newS3CompatibleProvider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "S3Compatible",
		CommandUse:  "s3-compatible",
		DisplayName: "S3 Compatible",
		Description: "Provision a generic S3-compatible repository (MinIO, Cloudian, Wasabi, ...).",
		newConfigurator: func() objectRepositoryConfigurator {
			return &s3CompatibleConfigurator{
				connectionType:    "Direct",
				mountServerVPower: true,
			}
		},
	}
}

func (c *s3CompatibleConfigurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.credentialsID, "credentials-id", "", "Cloud credentials ID for the S3-compatible endpoint")
	flags.StringVar(&c.servicePoint, "service-point", "", "Service endpoint URL (for example https://minio.local:9000)")
	flags.StringVar(&c.regionID, "region-id", "", "Region or location identifier reported by the provider")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.bucketName, "bucket-name", "", "Target bucket name")
	flags.StringVar(&c.folderName, "folder", "", "Folder/prefix inside the bucket")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", c.mountServerVPower, "Enable vPower NFS on the mount server")
	flags.BoolVar(&c.immutabilityOn, "immutability-enabled", false, "Enable immutability for stored backups")
	flags.IntVar(&c.immutabilityDays, "immutability-days", 0, "Immutability retention period in days")

	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *s3CompatibleConfigurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultS3CompatibleSpec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "S3Compatible"); err != nil {
		return nil, err
	}

	if flags.Changed("credentials-id") {
		if strings.TrimSpace(c.credentialsID) == "" {
			return nil, fmt.Errorf("--credentials-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "credentialsId"}, strings.TrimSpace(c.credentialsID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("service-point") {
		if strings.TrimSpace(c.servicePoint) == "" {
			return nil, fmt.Errorf("--service-point cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "servicePoint"}, strings.TrimSpace(c.servicePoint)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("region-id") {
		if strings.TrimSpace(c.regionID) == "" {
			return nil, fmt.Errorf("--region-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "regionId"}, strings.TrimSpace(c.regionID)); err != nil {
			return nil, err
		}
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, trimSlice(c.gatewayIDs)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("bucket-name") {
		if err := setNestedField(payload, []string{"bucket", "bucketName"}, strings.TrimSpace(c.bucketName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"bucket", "folderName"}, strings.TrimSpace(c.folderName)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("immutability-enabled") {
		if err := setNestedField(payload, []string{"bucket", "immutability", "isEnabled"}, c.immutabilityOn); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-days") {
		if c.immutabilityDays < 0 {
			return nil, fmt.Errorf("--immutability-days cannot be negative")
		}
		if err := setNestedField(payload, []string{"bucket", "immutability", "daysCount"}, c.immutabilityDays); err != nil {
			return nil, err
		}
	}

	if flags.Changed("mount-server-id") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-cache") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
			return nil, err
		}
	}
	if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
		return nil, err
	}

	if err := ensureString([]string{"account", "credentialsId"}, "credentials-id", "cloud credential ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"account", "servicePoint"}, "service-point", "service endpoint", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"account", "regionId"}, "region-id", "region identifier", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "bucketName"}, "bucket-name", "bucket name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "folderName"}, "folder", "folder name", payload); err != nil {
		return nil, err
	}
	if connectionType == "SelectedGateway" {
		if len(nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}
	if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func defaultS3CompatibleSpec() map[string]any {
	return map[string]any{
		"type":        "S3Compatible",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"servicePoint": "",
			"regionId":     "",
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"bucket": map[string]any{
			"bucketName": "",
			"folderName": "",
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}

// ------------------- Wasabi Cloud -------------------

type wasabiCloudConfigurator struct {
	credentialsID     string
	regionID          string
	connectionType    string
	gatewayIDs        []string
	bucketName        string
	folderName        string
	mountServerID     string
	mountServerCache  string
	mountServerVPower bool
	immutabilityOn    bool
	immutabilityDays  int
}

func newWasabiCloudProvider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "WasabiCloud",
		CommandUse:  "wasabi-cloud",
		DisplayName: "Wasabi Cloud",
		Description: "Provision a Wasabi S3-compatible object repository.",
		newConfigurator: func() objectRepositoryConfigurator {
			return &wasabiCloudConfigurator{
				connectionType:    "Direct",
				mountServerVPower: true,
			}
		},
		Aliases: []string{"wasabi"},
	}
}

func (c *wasabiCloudConfigurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.credentialsID, "credentials-id", "", "Cloud credentials ID for Wasabi access")
	flags.StringVar(&c.regionID, "region-id", "", "Wasabi region identifier (for example us-east-1)")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.bucketName, "bucket-name", "", "Target bucket name")
	flags.StringVar(&c.folderName, "folder", "", "Folder/prefix inside the bucket")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", c.mountServerVPower, "Enable vPower NFS on the mount server")
	flags.BoolVar(&c.immutabilityOn, "immutability-enabled", false, "Enable immutability for stored backups")
	flags.IntVar(&c.immutabilityDays, "immutability-days", 0, "Immutability retention period in days")

	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *wasabiCloudConfigurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultWasabiCloudSpec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "WasabiCloud"); err != nil {
		return nil, err
	}

	if flags.Changed("credentials-id") {
		if strings.TrimSpace(c.credentialsID) == "" {
			return nil, fmt.Errorf("--credentials-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "credentialsId"}, strings.TrimSpace(c.credentialsID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("region-id") {
		if strings.TrimSpace(c.regionID) == "" {
			return nil, fmt.Errorf("--region-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "regionId"}, strings.TrimSpace(c.regionID)); err != nil {
			return nil, err
		}
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, trimSlice(c.gatewayIDs)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("bucket-name") {
		if err := setNestedField(payload, []string{"bucket", "bucketName"}, strings.TrimSpace(c.bucketName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"bucket", "folderName"}, strings.TrimSpace(c.folderName)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("immutability-enabled") {
		if err := setNestedField(payload, []string{"bucket", "immutability", "isEnabled"}, c.immutabilityOn); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-days") {
		if c.immutabilityDays < 0 {
			return nil, fmt.Errorf("--immutability-days cannot be negative")
		}
		if err := setNestedField(payload, []string{"bucket", "immutability", "daysCount"}, c.immutabilityDays); err != nil {
			return nil, err
		}
	}

	if flags.Changed("mount-server-id") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-cache") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
			return nil, err
		}
	}
	if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
		return nil, err
	}

	if err := ensureString([]string{"account", "credentialsId"}, "credentials-id", "cloud credential ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"account", "regionId"}, "region-id", "region identifier", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "bucketName"}, "bucket-name", "bucket name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"bucket", "folderName"}, "folder", "folder name", payload); err != nil {
		return nil, err
	}
	if connectionType == "SelectedGateway" {
		if len(nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}
	if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func defaultWasabiCloudSpec() map[string]any {
	return map[string]any{
		"type":        "WasabiCloud",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"regionId": "",
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"bucket": map[string]any{
			"bucketName": "",
			"folderName": "",
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}

// ------------------- Azure Blob -------------------

type azureBlobConfigurator struct {
	credentialsID     string
	regionScope       string
	connectionType    string
	gatewayIDs        []string
	containerName     string
	folderName        string
	mountServerID     string
	mountServerCache  string
	mountServerVPower bool
	immutabilityOn    bool
	immutabilityDays  int
}

func newAzureBlobProvider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "AzureBlob",
		CommandUse:  "azure-blob",
		DisplayName: "Azure Blob",
		Description: "Create an Azure Blob storage repository with friendly flags.",
		newConfigurator: func() objectRepositoryConfigurator {
			return &azureBlobConfigurator{
				regionScope:       "Global",
				connectionType:    "Direct",
				mountServerVPower: true,
			}
		},
		Aliases: []string{"azure"},
	}
}

func (c *azureBlobConfigurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.credentialsID, "credentials-id", "", "Azure storage credentials ID")
	flags.StringVar(&c.regionScope, "region-scope", c.regionScope, "Azure cloud (Global|China|Government)")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.containerName, "container-name", "", "Azure Blob container name")
	flags.StringVar(&c.folderName, "folder", "", "Folder/prefix inside the container")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", c.mountServerVPower, "Enable vPower NFS on the mount server")
	flags.BoolVar(&c.immutabilityOn, "immutability-enabled", false, "Enable immutability for stored backups")
	flags.IntVar(&c.immutabilityDays, "immutability-days", 0, "Immutability retention period in days")

	_ = cmd.RegisterFlagCompletionFunc("region-scope", fixedCompletion(s3RegionScopes))
	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *azureBlobConfigurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultAzureBlobSpec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "AzureBlob"); err != nil {
		return nil, err
	}

	if flags.Changed("credentials-id") {
		if strings.TrimSpace(c.credentialsID) == "" {
			return nil, fmt.Errorf("--credentials-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "credentialsId"}, strings.TrimSpace(c.credentialsID)); err != nil {
			return nil, err
		}
	}

	currentRegionType := nestedString(payload, "account", "regionType")
	if flags.Changed("region-scope") || strings.TrimSpace(currentRegionType) == "" {
		normalized, normErr := normalizeEnum(c.regionScope, s3RegionScopes, "region-scope")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "regionType"}, normalized); err != nil {
			return nil, err
		}
	} else if _, normErr := normalizeEnum(currentRegionType, s3RegionScopes, "account.regionType"); normErr != nil {
		return nil, normErr
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, trimSlice(c.gatewayIDs)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("container-name") {
		if err := setNestedField(payload, []string{"container", "containerName"}, strings.TrimSpace(c.containerName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"container", "folderName"}, strings.TrimSpace(c.folderName)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("immutability-enabled") {
		if err := setNestedField(payload, []string{"container", "immutability", "isEnabled"}, c.immutabilityOn); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-days") {
		if c.immutabilityDays < 0 {
			return nil, fmt.Errorf("--immutability-days cannot be negative")
		}
		if err := setNestedField(payload, []string{"container", "immutability", "daysCount"}, c.immutabilityDays); err != nil {
			return nil, err
		}
	}

	if flags.Changed("mount-server-id") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-cache") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
			return nil, err
		}
	}
	if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
		return nil, err
	}

	if err := ensureString([]string{"account", "credentialsId"}, "credentials-id", "cloud credential ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"account", "regionType"}, "region-scope", "Azure region scope", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"container", "containerName"}, "container-name", "container name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"container", "folderName"}, "folder", "folder name", payload); err != nil {
		return nil, err
	}
	if connectionType == "SelectedGateway" {
		if len(nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}
	if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func defaultAzureBlobSpec() map[string]any {
	return map[string]any{
		"type":        "AzureBlob",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"regionType": "Global",
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"container": map[string]any{
			"containerName": "",
			"folderName":    "",
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}

// ------------------- Azure Archive -------------------

type azureArchiveConfigurator struct {
	credentialsID       string
	regionScope         string
	connectionType      string
	gatewayIDs          []string
	containerName       string
	folderName          string
	immutabilityEnabled bool
	proxySubscriptionID string
	proxyResourceGroup  string
	proxyVirtualNetwork string
	proxySubnet         string
	proxyInstanceSize   string
	proxyRedirectorPort int
	mountServerID       string
	mountServerCache    string
	mountServerVPower   bool
}

func newAzureArchiveProvider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "AzureArchive",
		CommandUse:  "azure-archive",
		DisplayName: "Azure Archive",
		Description: "Create an Azure Archive object repository with helper flags.",
		newConfigurator: func() objectRepositoryConfigurator {
			return &azureArchiveConfigurator{
				regionScope:         "Global",
				connectionType:      "Direct",
				immutabilityEnabled: false,
			}
		},
	}
}

func (c *azureArchiveConfigurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.credentialsID, "credentials-id", "", "Azure storage credentials ID")
	flags.StringVar(&c.regionScope, "region-scope", c.regionScope, "Azure cloud (Global|China|Government)")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.containerName, "container-name", "", "Azure Archive container name")
	flags.StringVar(&c.folderName, "folder", "", "Folder/prefix inside the container")
	flags.BoolVar(&c.immutabilityEnabled, "immutability-enabled", c.immutabilityEnabled, "Enable immutability on the container")
	flags.StringVar(&c.proxySubscriptionID, "proxy-subscription-id", "", "Subscription ID for the proxy appliance")
	flags.StringVar(&c.proxyResourceGroup, "proxy-resource-group", "", "Resource group for the proxy appliance")
	flags.StringVar(&c.proxyVirtualNetwork, "proxy-virtual-network", "", "Virtual network for the proxy appliance")
	flags.StringVar(&c.proxySubnet, "proxy-subnet", "", "Subnet for the proxy appliance")
	flags.StringVar(&c.proxyInstanceSize, "proxy-instance-size", "", "Instance size for the proxy appliance")
	flags.IntVar(&c.proxyRedirectorPort, "proxy-redirector-port", 0, "Redirector TCP port for the proxy appliance")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", true, "Enable vPower NFS on the mount server")

	_ = cmd.RegisterFlagCompletionFunc("region-scope", fixedCompletion(s3RegionScopes))
	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *azureArchiveConfigurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultAzureArchiveSpec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "AzureArchive"); err != nil {
		return nil, err
	}

	if flags.Changed("credentials-id") {
		if strings.TrimSpace(c.credentialsID) == "" {
			return nil, fmt.Errorf("--credentials-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "credentialsId"}, strings.TrimSpace(c.credentialsID)); err != nil {
			return nil, err
		}
	}

	currentRegionType := nestedString(payload, "account", "regionType")
	if flags.Changed("region-scope") || strings.TrimSpace(currentRegionType) == "" {
		normalized, normErr := normalizeEnum(c.regionScope, s3RegionScopes, "region-scope")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "regionType"}, normalized); err != nil {
			return nil, err
		}
	} else if _, normErr := normalizeEnum(currentRegionType, s3RegionScopes, "account.regionType"); normErr != nil {
		return nil, normErr
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, trimSlice(c.gatewayIDs)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("container-name") {
		if err := setNestedField(payload, []string{"container", "containerName"}, strings.TrimSpace(c.containerName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"container", "folderName"}, strings.TrimSpace(c.folderName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-enabled") {
		if err := setNestedField(payload, []string{"container", "immutabilityEnabled"}, c.immutabilityEnabled); err != nil {
			return nil, err
		}
	}

	if flags.Changed("proxy-subscription-id") {
		if strings.TrimSpace(c.proxySubscriptionID) == "" {
			return nil, fmt.Errorf("--proxy-subscription-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"proxyAppliance", "subscriptionId"}, strings.TrimSpace(c.proxySubscriptionID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("proxy-resource-group") {
		if err := setNestedField(payload, []string{"proxyAppliance", "resourceGroup"}, strings.TrimSpace(c.proxyResourceGroup)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("proxy-virtual-network") {
		if err := setNestedField(payload, []string{"proxyAppliance", "virtualNetwork"}, strings.TrimSpace(c.proxyVirtualNetwork)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("proxy-subnet") {
		if err := setNestedField(payload, []string{"proxyAppliance", "subnet"}, strings.TrimSpace(c.proxySubnet)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("proxy-instance-size") {
		if err := setNestedField(payload, []string{"proxyAppliance", "instanceSize"}, strings.TrimSpace(c.proxyInstanceSize)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("proxy-redirector-port") {
		if c.proxyRedirectorPort < 0 {
			return nil, fmt.Errorf("--proxy-redirector-port cannot be negative")
		}
		if err := setNestedField(payload, []string{"proxyAppliance", "redirectorPort"}, c.proxyRedirectorPort); err != nil {
			return nil, err
		}
	}

	if flags.Changed("mount-server-id") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-cache") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
			return nil, err
		}
	}
	if valueExists(payload, "mountServer") {
		if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
			return nil, err
		}
	}

	if err := ensureString([]string{"account", "credentialsId"}, "credentials-id", "cloud credential ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"account", "regionType"}, "region-scope", "Azure region scope", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"container", "containerName"}, "container-name", "container name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"container", "folderName"}, "folder", "folder name", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"proxyAppliance", "subscriptionId"}, "proxy-subscription-id", "proxy subscription ID", payload); err != nil {
		return nil, err
	}
	if connectionType == "SelectedGateway" {
		if len(nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}
	if valueExists(payload, "mountServer") {
		if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
			return nil, err
		}
		if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
			return nil, err
		}
		if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func defaultAzureArchiveSpec() map[string]any {
	return map[string]any{
		"type":        "AzureArchive",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"regionType": "Global",
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"container": map[string]any{
			"containerName":       "",
			"folderName":          "",
			"immutabilityEnabled": false,
		},
		"proxyAppliance": map[string]any{
			"subscriptionId": "",
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}

// ------------------- Veeam Data Cloud Vault -------------------

type veeamDataCloudVaultConfigurator struct {
	vaultID           string
	vaultName         string
	vaultInitialized  bool
	connectionType    string
	gatewayIDs        []string
	folder            string
	immutabilityOn    bool
	immutabilityDays  int
	mountServerID     string
	mountServerCache  string
	mountServerVPower bool
}

func newVeeamDataCloudVaultProvider() objectRepositoryProvider {
	return objectRepositoryProvider{
		Canonical:   "VeeamDataCloudVault",
		CommandUse:  "veeam-data-cloud-vault",
		DisplayName: "Veeam Data Cloud Vault",
		Description: "Connect to a managed Veeam Data Cloud Vault repository.",
		newConfigurator: func() objectRepositoryConfigurator {
			return &veeamDataCloudVaultConfigurator{
				vaultInitialized:  true,
				connectionType:    "Direct",
				immutabilityOn:    true,
				mountServerVPower: true,
			}
		},
		Aliases: []string{"vault"},
	}
}

func (c *veeamDataCloudVaultConfigurator) BindFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&c.vaultID, "vault-id", "", "Veeam Data Cloud vault identifier")
	flags.StringVar(&c.vaultName, "vault-name", "", "Friendly name for the vault (optional)")
	flags.BoolVar(&c.vaultInitialized, "vault-initialized", c.vaultInitialized, "Mark the vault as already associated with this backup server")
	flags.StringVar(&c.connectionType, "connection-type", c.connectionType, "Connection mode (Direct|SelectedGateway)")
	flags.StringSliceVar(&c.gatewayIDs, "gateway-id", nil, "Gateway server ID for proxy access (repeatable)")
	flags.StringVar(&c.folder, "folder", "", "Folder name inside the vault")
	flags.BoolVar(&c.immutabilityOn, "immutability-enabled", c.immutabilityOn, "Enable immutability for stored backups")
	flags.IntVar(&c.immutabilityDays, "immutability-days", 0, "Immutability retention period in days (optional)")
	flags.StringVar(&c.mountServerID, "mount-server-id", "", "Mount server ID used for Instant Recovery operations")
	flags.StringVar(&c.mountServerCache, "mount-server-cache", "", "Mount server cache folder path")
	flags.BoolVar(&c.mountServerVPower, "mount-server-vpower", c.mountServerVPower, "Enable vPower NFS on the mount server")

	_ = cmd.RegisterFlagCompletionFunc("connection-type", fixedCompletion(repoConnectionTypes))
}

func (c *veeamDataCloudVaultConfigurator) BuildBase(cmd *cobra.Command, opts *objectRepositoryAddOptions) (map[string]any, error) {
	var payload map[string]any
	var err error
	if strings.TrimSpace(opts.specPath) != "" {
		payload, err = loadSpecFile(opts.specPath)
		if err != nil {
			return nil, err
		}
	} else {
		payload = defaultVeeamDataCloudVaultSpec()
	}

	flags := cmd.Flags()

	if err := setNestedField(payload, []string{"type"}, "VeeamDataCloudVault"); err != nil {
		return nil, err
	}

	if flags.Changed("vault-id") {
		if strings.TrimSpace(c.vaultID) == "" {
			return nil, fmt.Errorf("--vault-id cannot be empty")
		}
		if err := setNestedField(payload, []string{"account", "vault", "vaultId"}, strings.TrimSpace(c.vaultID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("vault-name") {
		if err := setNestedField(payload, []string{"account", "vault", "vaultName"}, strings.TrimSpace(c.vaultName)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("vault-initialized") || !valueExists(payload, "account", "vault", "isInitialized") {
		if err := setNestedField(payload, []string{"account", "vault", "isInitialized"}, c.vaultInitialized); err != nil {
			return nil, err
		}
	}

	connectionType := c.connectionType
	if strings.TrimSpace(connectionType) == "" {
		connectionType = "Direct"
	}
	if flags.Changed("connection-type") || strings.TrimSpace(nestedString(payload, "account", "connectionSettings", "connectionType")) == "" {
		normalized, normErr := normalizeEnum(connectionType, repoConnectionTypes, "connection-type")
		if normErr != nil {
			return nil, normErr
		}
		if err := setNestedField(payload, []string{"account", "connectionSettings", "connectionType"}, normalized); err != nil {
			return nil, err
		}
		connectionType = normalized
	} else {
		existing := nestedString(payload, "account", "connectionSettings", "connectionType")
		if existing != "" {
			if normalized, normErr := normalizeEnum(existing, repoConnectionTypes, "account.connectionSettings.connectionType"); normErr != nil {
				return nil, normErr
			} else {
				connectionType = normalized
			}
		}
	}

	if flags.Changed("gateway-id") {
		if err := setNestedField(payload, []string{"account", "connectionSettings", "gatewayServerIds"}, trimSlice(c.gatewayIDs)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("folder") {
		if err := setNestedField(payload, []string{"container", "folder"}, strings.TrimSpace(c.folder)); err != nil {
			return nil, err
		}
	}

	if flags.Changed("immutability-enabled") || !valueExists(payload, "container", "immutability", "isEnabled") {
		if err := setNestedField(payload, []string{"container", "immutability", "isEnabled"}, c.immutabilityOn); err != nil {
			return nil, err
		}
	}
	if flags.Changed("immutability-days") {
		if c.immutabilityDays < 0 {
			return nil, fmt.Errorf("--immutability-days cannot be negative")
		}
		if err := setNestedField(payload, []string{"container", "immutability", "daysCount"}, c.immutabilityDays); err != nil {
			return nil, err
		}
	}

	if flags.Changed("mount-server-id") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "mountServerId"}, strings.TrimSpace(c.mountServerID)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-cache") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "writeCacheFolder"}, strings.TrimSpace(c.mountServerCache)); err != nil {
			return nil, err
		}
	}
	if flags.Changed("mount-server-vpower") || !valueExists(payload, "mountServer", "windows", "vPowerNFSEnabled") {
		if err := setNestedField(payload, []string{"mountServer", "windows", "vPowerNFSEnabled"}, c.mountServerVPower); err != nil {
			return nil, err
		}
	}
	if err := setNestedField(payload, []string{"mountServer", "mountServerSettingsType"}, "windows"); err != nil {
		return nil, err
	}

	if err := ensureString([]string{"account", "vault", "vaultId"}, "vault-id", "vault ID", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"account", "vault", "isInitialized"}, "vault-initialized", "vault initialization flag", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"container", "folder"}, "folder", "vault folder", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"container", "immutability", "isEnabled"}, "immutability-enabled", "immutability toggle", payload); err != nil {
		return nil, err
	}
	if connectionType == "SelectedGateway" {
		if len(nestedStringSlice(payload, "account", "connectionSettings", "gatewayServerIds")) == 0 {
			return nil, fmt.Errorf("connection type SelectedGateway requires at least one gateway server (--gateway-id)")
		}
	}
	if err := ensureString([]string{"mountServer", "windows", "mountServerId"}, "mount-server-id", "mount server ID", payload); err != nil {
		return nil, err
	}
	if err := ensureString([]string{"mountServer", "windows", "writeCacheFolder"}, "mount-server-cache", "mount server cache", payload); err != nil {
		return nil, err
	}
	if err := ensureBool([]string{"mountServer", "windows", "vPowerNFSEnabled"}, "mount-server-vpower", "vPower NFS toggle", payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func defaultVeeamDataCloudVaultSpec() map[string]any {
	return map[string]any{
		"type":        "VeeamDataCloudVault",
		"name":        "",
		"description": "",
		"isDisabled":  false,
		"account": map[string]any{
			"vault": map[string]any{
				"vaultId":       "",
				"vaultName":     "",
				"isInitialized": true,
			},
			"connectionSettings": map[string]any{
				"connectionType":   "Direct",
				"gatewayServerIds": []string{},
			},
		},
		"container": map[string]any{
			"folder": "",
			"immutability": map[string]any{
				"isEnabled": true,
			},
		},
		"mountServer": map[string]any{
			"mountServerSettingsType": "windows",
			"windows": map[string]any{
				"mountServerId":    "",
				"writeCacheFolder": "",
				"vPowerNFSEnabled": true,
			},
		},
	}
}
