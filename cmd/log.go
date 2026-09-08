package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/journal"
	"github.com/eggylan/recall-loom-lite/internal/storage"
)

func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Daily log operations",
	}
	cmd.AddCommand(newLogAppendCmd())
	return cmd
}

func newLogAppendCmd() *cobra.Command {
	var (
		title  string
		date   string
		ts     string
		dryRun bool
		file   string
	)
	cmd := &cobra.Command{
		Use:   "append --title TITLE",
		Short: "Append a milestone entry to the daily log (body from stdin or --file)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readInput(cmd, file)
			if err != nil {
				return err
			}
			now := time.Now()
			if date == "" {
				date = now.Format("2006-01-02")
			}
			if ts == "" {
				ts = now.Format("15:04")
			}
			e := journal.Entry{Time: ts, Title: title, Body: body}
			if dryRun {
				fmt.Fprint(cmd.OutOrStdout(), journal.RenderEntry(e))
				return nil
			}
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			path, err := journal.Append(root, date, e)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "appended to %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "entry title (required)")
	cmd.Flags().StringVar(&date, "date", "", "log date YYYY-MM-DD (default: today, local time)")
	cmd.Flags().StringVar(&ts, "time", "", "entry time HH:MM (default: now, local time)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the rendered entry instead of writing")
	cmd.Flags().StringVar(&file, "file", "", "read entry body from this file (default: stdin)")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}
