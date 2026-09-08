package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show a one-screen overview of the project memory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, ".rll root: %s\n", root)

			describe := func(name string, required bool) {
				p := filepath.Join(root, name)
				fi, err := os.Stat(p)
				if os.IsNotExist(err) {
					if required {
						fmt.Fprintf(out, "%s: MISSING\n", name)
					} else {
						fmt.Fprintf(out, "%s: absent\n", name)
					}
					return
				}
				if err != nil {
					fmt.Fprintf(out, "%s: unreadable: %v\n", name, err)
					return
				}
				fmt.Fprintf(out, "%s: %d bytes, modified %s\n", name, fi.Size(), fi.ModTime().Format("2006-01-02 15:04"))
			}
			describe("context_brief.md", true)
			describe("rolling_summary.md", true)
			describe("update_protocol.md", false)

			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			archived, err := journal.ListArchive(root)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "daily logs: %d active, %d archived\n", len(logs), len(archived))
			if latest, _ := journal.Latest(root); latest != nil {
				entries, err := journal.ReadEntries(*latest)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "latest log: %s (%d entries)\n", latest.Date, len(entries))
			} else {
				fmt.Fprintln(out, "latest log: none")
			}
			return nil
		},
	}
}
