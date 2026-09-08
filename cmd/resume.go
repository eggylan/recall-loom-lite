package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

type resumeJSON struct {
	LatestLogDate  string          `json:"latest_log_date,omitempty"`
	EntryCount     int             `json:"entry_count"`
	UpdateProtocol string          `json:"update_protocol,omitempty"`
	Brief          string          `json:"brief"`
	Summary        string          `json:"summary"`
	LatestLog      []journal.Entry `json:"latest_log,omitempty"`
}

func newResumeCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Print the full project memory context for a cold start (one tier: everything)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			read := func(name string) (string, bool) {
				data, err := os.ReadFile(filepath.Join(root, name))
				if err != nil {
					return "", false
				}
				return string(data), true
			}

			latest, err := journal.Latest(root)
			if err != nil {
				return err
			}
			var entries []journal.Entry
			var latestDate string
			if latest != nil {
				entries, err = journal.ReadEntries(*latest)
				if err != nil {
					return err
				}
				latestDate = latest.Date
			}
			protocol, hasProtocol := read("update_protocol.md")
			brief, _ := read("context_brief.md")
			summary, _ := read("rolling_summary.md")

			if asJSON {
				out := resumeJSON{
					LatestLogDate:  latestDate,
					EntryCount:     len(entries),
					UpdateProtocol: protocol,
					Brief:          brief,
					Summary:        summary,
					LatestLog:      entries,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "# rll resume")
			fmt.Fprintln(out)
			if latestDate != "" {
				fmt.Fprintf(out, "latest log: %s (%d entries)\n", latestDate, len(entries))
			} else {
				fmt.Fprintln(out, "latest log: none")
			}
			fmt.Fprintf(out, "protocol: %s\n", map[bool]string{true: "present", false: "absent"}[hasProtocol])
			if hasProtocol {
				fmt.Fprint(out, "\n## Update Protocol\n\n")
				fmt.Fprint(out, protocol)
			}
			fmt.Fprint(out, "\n## Context Brief\n\n")
			fmt.Fprint(out, brief)
			fmt.Fprint(out, "\n## Rolling Summary\n\n")
			fmt.Fprint(out, summary)
			fmt.Fprint(out, "\n## Daily Log: "+latestDate+"\n\n")
			if latest != nil {
				data, err := os.ReadFile(latest.Path)
				if err != nil {
					return err
				}
				fmt.Fprint(out, string(data))
			} else {
				fmt.Fprintln(out, "(no daily logs yet)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
