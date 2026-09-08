package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create the .rll sidecar skeleton in this project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rll := filepath.Join(cwd, storage.DirName)
			if err := storage.CheckInitTarget(rll); err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Join(rll, "daily_logs"), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(rll, "context_brief.md"), []byte(storage.RenderDocument(storage.BriefTemplate)), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(rll, "rolling_summary.md"), []byte(storage.RenderDocument(storage.SummaryTemplate)), 0o644); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "initialized %s\n", rll)
			if storage.IsGitRoot(cwd) {
				msg, err := storage.EnsureGitExclude(cwd)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: git exclude: %v\n", err)
				} else {
					fmt.Fprintf(out, "git: %s\n", msg)
				}
			} else {
				fmt.Fprintln(out, "git: skipped (current directory is not a git repository root)")
			}
			fmt.Fprintln(out, "next: fill context_brief.md and rolling_summary.md, e.g.")
			fmt.Fprintln(out, "  rll write brief   (body via stdin, no frontmatter needed)")
			fmt.Fprintln(out, "  rll write summary")
			return nil
		},
	}
}
