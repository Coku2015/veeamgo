package jobtemplates

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Variant identifies the shape of the template.
type Variant string

const (
	VariantMinimal Variant = "minimal"
	VariantFull    Variant = "full"
)

// TemplateInfo describes an available template entry.
type TemplateInfo struct {
	JobType     string
	Variant     Variant
	Description string
}

//go:embed data/*.yaml
var templateFS embed.FS

var registry = map[string]struct {
	Description string
	Files       map[Variant]string
}{
	"VSphereBackup": {
		Description: "VMware vSphere backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/vspherebackup_minimal.yaml",
			VariantFull:    "data/vspherebackup_full.yaml",
		},
	},
	"HyperVBackup": {
		Description: "Microsoft Hyper-V backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/hypervbackup_minimal.yaml",
			VariantFull:    "data/hypervbackup_full.yaml",
		},
	},
	"CloudDirectorBackup": {
		Description: "VMware Cloud Director backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/clouddirectorbackup_minimal.yaml",
			VariantFull:    "data/clouddirectorbackup_full.yaml",
		},
	},
	"VSphereReplica": {
		Description: "VMware vSphere replica job",
		Files: map[Variant]string{
			VariantMinimal: "data/vspherereplica_minimal.yaml",
			VariantFull:    "data/vspherereplica_full.yaml",
		},
	},
	"BackupCopy": {
		Description: "Backup copy job",
		Files: map[Variant]string{
			VariantMinimal: "data/backupcopy_minimal.yaml",
			VariantFull:    "data/backupcopy_full.yaml",
		},
	},
	"FileBackupCopy": {
		Description: "File backup copy job",
		Files: map[Variant]string{
			VariantMinimal: "data/filebackupcopy_minimal.yaml",
			VariantFull:    "data/filebackupcopy_full.yaml",
		},
	},
	"WindowsAgentBackup": {
		Description: "Windows agent backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/windowsagentbackup_minimal.yaml",
			VariantFull:    "data/windowsagentbackup_full.yaml",
		},
	},
	"LinuxAgentBackup": {
		Description: "Linux agent backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/linuxagentbackup_minimal.yaml",
			VariantFull:    "data/linuxagentbackup_full.yaml",
		},
	},
	"EntraIDTenantBackup": {
		Description: "Microsoft Entra ID backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/entraidtenantbackup_minimal.yaml",
			VariantFull:    "data/entraidtenantbackup_full.yaml",
		},
	},
	"EntraIDAuditLogBackup": {
		Description: "Microsoft Entra ID audit log backup job",
		Files: map[Variant]string{
			VariantMinimal: "data/entraidtenantauditlogbackup_minimal.yaml",
			VariantFull:    "data/entraidtenantauditlogbackup_full.yaml",
		},
	},
	"EntraIDTenantBackupCopy": {
		Description: "Microsoft Entra ID backup copy job",
		Files: map[Variant]string{
			VariantMinimal: "data/entraidtenantbackupcopy_minimal.yaml",
			VariantFull:    "data/entraidtenantbackupcopy_full.yaml",
		},
	},
}

var aliasIndex map[string]string

func init() {
	aliasIndex = make(map[string]string)
	for jobType := range registry {
		aliasIndex[strings.ToLower(jobType)] = jobType
	}
}

// SupportedJobTypes returns the list of job types with available templates.
func SupportedJobTypes() []string {
	types := make([]string, 0, len(registry))
	for jobType := range registry {
		types = append(types, jobType)
	}
	sort.Strings(types)
	return types
}

// VariantsFor returns the supported variants for a job type.
func VariantsFor(jobType string) []Variant {
	entry, ok := registry[normalizeJobType(jobType)]
	if !ok {
		return nil
	}
	variants := make([]Variant, 0, len(entry.Files))
	for variant := range entry.Files {
		variants = append(variants, variant)
	}
	sort.Slice(variants, func(i, j int) bool { return variants[i] < variants[j] })
	return variants
}

// Render returns the YAML template for the given job type and variant.
func Render(jobType string, variant Variant, includeComments bool) ([]byte, error) {
	path, err := resolvePath(jobType, variant)
	if err != nil {
		return nil, err
	}
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}
	if includeComments {
		return data, nil
	}
	lines := strings.Split(string(data), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		filtered = append(filtered, line)
	}
	return []byte(strings.Join(filtered, "\n")), nil
}

// Spec decodes the template into a map suitable for blueprint seeding.
func Spec(jobType string, variant Variant) (map[string]any, error) {
	data, err := Render(jobType, variant, true)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode template: %w", err)
	}
	return normalizeMap(raw), nil
}

// List returns all available templates with display metadata.
func List() []TemplateInfo {
	infos := make([]TemplateInfo, 0)
	for jobType, entry := range registry {
		for variant := range entry.Files {
			infos = append(infos, TemplateInfo{
				JobType:     jobType,
				Variant:     variant,
				Description: entry.Description,
			})
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].JobType == infos[j].JobType {
			return infos[i].Variant < infos[j].Variant
		}
		return infos[i].JobType < infos[j].JobType
	})
	return infos
}

// NormalizeVariant canonicalises user-provided variants.
func NormalizeVariant(value string) Variant {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(VariantMinimal):
		return VariantMinimal
	case string(VariantFull):
		return VariantFull
	default:
		return Variant(strings.TrimSpace(value))
	}
}

func resolvePath(jobType string, variant Variant) (string, error) {
	canonical := normalizeJobType(jobType)
	entry, ok := registry[canonical]
	if !ok {
		return "", fmt.Errorf("no template available for job type %q", jobType)
	}
	if path, ok := entry.Files[variant]; ok {
		return path, nil
	}
	if variant == VariantFull {
		return "", fmt.Errorf("variant %q not available for job type %q", variant, jobType)
	}
	return "", fmt.Errorf("variant %q not available for job type %q", variant, jobType)
}

func normalizeJobType(value string) string {
	if value == "" {
		return ""
	}
	if canonical, ok := aliasIndex[strings.ToLower(strings.TrimSpace(value))]; ok {
		return canonical
	}
	return value
}

func normalizeMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return convertMap(input)
}

func convertMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		for key, inner := range typed {
			typed[key] = convertValue(inner)
		}
		return typed
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, inner := range typed {
			keyStr := fmt.Sprint(key)
			result[keyStr] = convertValue(inner)
		}
		return result
	default:
		return map[string]any{}
	}
}

func convertValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, inner := range typed {
			typed[key] = convertValue(inner)
		}
		return typed
	case map[any]any:
		result := make(map[string]any, len(typed))
		for key, inner := range typed {
			result[fmt.Sprint(key)] = convertValue(inner)
		}
		return result
	case []any:
		for i, inner := range typed {
			typed[i] = convertValue(inner)
		}
		return typed
	default:
		return value
	}
}
