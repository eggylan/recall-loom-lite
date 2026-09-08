package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newArchiveCmd() *cobra.Command {
	var (
		before string
		apply  bool
	)
	cmd := &cobra.Command{
		Use:   "archive --before YYYY-MM-DD [--apply]",
		Short: "Move daily logs older than a date into .rll/archive (preview by default)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !journal.ValidDateFileName(before + ".md") {
				return fmt.Errorf("--before is required and must be YYYY-MM-DD")
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			var toMove []journal.LogFile
			for _, l := range logs {
				if l.Date < before {
					toMove = append(toMove, l)
				}
			}
			out := cmd.OutOrStdout()
			if len(toMove) == 0 {
				fmt.Fprintln(out, "nothing to archive")
				return nil
			}
			if !apply {
				for _, l := range toMove {
					fmt.Fprintf(out, "would move: %s/%s.md -> %s/%s.md\n",
						journal.LogDirName, l.Date, journal.ArchiveDirName, l.Date)
				}
				fmt.Fprintf(out, "total: %d file(s) (use --apply to move)\n", len(toMove))
				return nil
			}
			dstDir := filepath.Join(root, journal.ArchiveDirName)
			if err := os.MkdirAll(dstDir, 0o755); err != nil {
				return err
			}
			for _, l := range toMove {
				dst := filepath.Join(dstDir, l.Date+".md")
				if _, err := os.Stat(dst); err == nil {
					return fmt.Errorf("archive target already exists: %s", dst)
				}
				if err := os.Rename(l.Path, dst); err != nil {
					return err
				}
				fmt.Fprintf(out, "moved: %s -> %s\n", l.Path, dst)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&before, "before", "", "archive logs with dates strictly before this YYYY-MM-DD (required)")
	cmd.Flags().BoolVar(&apply, "apply", false, "actually move files (default: preview only)")
	return cmd
}
