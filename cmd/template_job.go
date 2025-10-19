package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	jobtemplates "github.com/veeamgo/veeamgo/internal/templates/job"
	"github.com/veeamgo/veeamgo/pkg/output"
)

func templateJobCmd() *cobra.Command {
	opts := struct {
		jobType      string
		variant      string
		writePath    string
		listOnly     bool
		stripComment bool
	}{
		variant: string(jobtemplates.VariantMinimal),
	}

	cmd := &cobra.Command{
		Use:   cmdJobUse,
		Short: "Export job configuration templates",
		Long:  "Generate YAML blueprints for creating or editing jobs. Combine with 'veeamgo add job --template' to seed new jobs quickly.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.listOnly {
				return renderTemplateList(cmd)
			}
			if strings.TrimSpace(opts.jobType) == "" {
				return fmt.Errorf("provide --type or use --list to view available templates")
			}

			variant := jobtemplates.NormalizeVariant(opts.variant)
			if !variantSupported(opts.jobType, variant) {
				available := jobtemplates.VariantsFor(opts.jobType)
				if len(available) == 0 {
					return fmt.Errorf("job type %q is not supported; see 'veeamgo template job --list'", opts.jobType)
				}
				return fmt.Errorf("variant %q is not available for %q (supported: %s)", variant, opts.jobType, joinVariants(available))
			}

			includeComments := !opts.stripComment
			payload, err := jobtemplates.Render(opts.jobType, variant, includeComments)
			if err != nil {
				return err
			}

			if strings.TrimSpace(opts.writePath) == "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			}

			resolved := filepath.Clean(opts.writePath)
			if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
				return fmt.Errorf("ensure destination directory: %w", err)
			}
			if err := os.WriteFile(resolved, ensureTrailingNewline(payload), 0o600); err != nil {
				return fmt.Errorf("write template: %w", err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Wrote template to %s\n", resolved)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.jobType, "type", "", "Job type (EJobType value, see --list)")
	cmd.Flags().StringVar(&opts.variant, "variant", string(jobtemplates.VariantMinimal), "Template variant (minimal|full)")
	cmd.Flags().StringVar(&opts.writePath, "write", "", "Destination file path (omit to print to stdout)")
	cmd.Flags().BoolVar(&opts.listOnly, "list", false, "List available job templates")
	cmd.Flags().BoolVar(&opts.stripComment, "no-comments", false, "Omit comment lines from the generated template")

	return cmd
}

func renderTemplateList(cmd *cobra.Command) error {
	infos := jobtemplates.List()
	if len(infos) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No templates available.")
		return nil
	}

	type templateRow struct {
		JobType     string `json:"Job Type"`
		Variants    string `json:"Variants"`
		Description string `json:"Description"`
	}

	rows := make([]templateRow, 0, len(infos))
	grouped := make(map[string][]string)
	descriptions := make(map[string]string)
	for _, info := range infos {
		grouped[info.JobType] = append(grouped[info.JobType], string(info.Variant))
		if _, ok := descriptions[info.JobType]; !ok {
			descriptions[info.JobType] = info.Description
		}
	}

	types := make([]string, 0, len(grouped))
	for jobType := range grouped {
		types = append(types, jobType)
	}
	sort.Strings(types)
	for _, jobType := range types {
		variants := grouped[jobType]
		sort.Strings(variants)
		rows = append(rows, templateRow{
			JobType:     jobType,
			Variants:    strings.Join(variants, ", "),
			Description: descriptions[jobType],
		})
	}

	return output.Print(outputFormat(), rows)
}

func variantSupported(jobType string, variant jobtemplates.Variant) bool {
	for _, candidate := range jobtemplates.VariantsFor(jobType) {
		if candidate == variant {
			return true
		}
	}
	return false
}

func joinVariants(variants []jobtemplates.Variant) string {
	if len(variants) == 0 {
		return ""
	}
	parts := make([]string, len(variants))
	for i, variant := range variants {
		parts[i] = string(variant)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] == '\n' {
		return data
	}
	return append(data, '\n')
}
