package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

var writeTargets = map[string]string{
	"brief":    "context_brief.md",
	"summary":  "rolling_summary.md",
	"protocol": "update_protocol.md",
}

func newWriteCmd() *cobra.Command {
	var (
		dryRun bool
		file   string
	)
	cmd := &cobra.Command{
		Use:       "write <brief|summary|protocol>",
		Short:     "Overwrite a managed document (body from stdin or --file, frontmatter added automatically)",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"brief", "summary", "protocol"},
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			body, err := readInput(cmd, file)
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("refusing to write empty content")
			}
			content := storage.RenderDocument(body)
			if dryRun {
				fmt.Fprint(cmd.OutOrStdout(), content)
				return nil
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			path := filepath.Join(root, writeTargets[target])
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the rendered document instead of writing")
	cmd.Flags().StringVar(&file, "file", "", "read body from this file (default: stdin)")
	return cmd
}
