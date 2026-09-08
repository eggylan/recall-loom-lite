package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newQueryCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "query <text...>",
		Short: "Case-insensitive substring search across rolling_summary and daily logs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			needle := strings.ToLower(strings.Join(args, " "))
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			matches := 0

			type source struct {
				rel  string
				path string
			}
			var sources []source
			sources = append(sources, source{rel: "rolling_summary.md", path: filepath.Join(root, "rolling_summary.md")})
			logs, err := journal.List(root)
			if err != nil {
				return err
			}
			for _, l := range logs {
				sources = append(sources, source{rel: journal.LogDirName + "/" + l.Date + ".md", path: l.Path})
			}
			if all {
				archived, err := journal.ListArchive(root)
				if err != nil {
					return err
				}
				for _, l := range archived {
					sources = append(sources, source{rel: journal.ArchiveDirName + "/" + l.Date + ".md", path: l.Path})
				}
			}

			for _, s := range sources {
				data, err := os.ReadFile(s.path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skip %s: %v\n", s.rel, err)
					continue
				}
				var printedHeader bool
				for i, line := range strings.Split(string(data), "\n") {
					if strings.Contains(strings.ToLower(line), needle) {
						if !printedHeader {
							fmt.Fprintf(out, "%s\n", s.rel)
							printedHeader = true
						}
						fmt.Fprintf(out, "  %d: %s\n", i+1, strings.TrimRight(line, "\r"))
						matches++
					}
				}
			}
			if matches == 0 {
				fmt.Fprintln(out, "no matches")
				os.Exit(1)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "also search archived daily logs")
	return cmd
}
