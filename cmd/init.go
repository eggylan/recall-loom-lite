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
			// Prefer the git repository root so .rll lives beside .git.
			target := cwd
			if root, ok := storage.GitRoot(cwd); ok {
				target = root
			}
			rll := filepath.Join(target, storage.DirName)
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
			if root, ok := storage.GitRoot(target); ok {
				msg, err := storage.EnsureGitExclude(root)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: git exclude: %v\n", err)
				} else {
					fmt.Fprintf(out, "git: %s\n", msg)
				}
			}
			fmt.Fprintln(out, "next: fill context_brief.md and rolling_summary.md, e.g.")
			fmt.Fprintln(out, "  rll write brief   (body via stdin, no frontmatter needed)")
			fmt.Fprintln(out, "  rll write summary")
			return nil
		},
	}
}
