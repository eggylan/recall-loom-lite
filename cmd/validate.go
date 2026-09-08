package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eggylan/recall-loom-lite/internal/storage"
	"github.com/eggylan/recall-loom-lite/internal/validate"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check .rll file formats (structure only, no semantics)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := storage.CwdRoot()
			if err != nil {
				return err
			}
			findings := validate.Run(root)
			out := cmd.OutOrStdout()
			errCount, warnCount := 0, 0
			for _, f := range findings {
				fmt.Fprintf(out, "%s: %s: %s\n", f.Severity, f.Path, f.Message)
				if f.Severity == validate.Error {
					errCount++
				} else {
					warnCount++
				}
			}
			fmt.Fprintf(out, "%d error(s), %d warning(s)\n", errCount, warnCount)
			if validate.HasErrors(findings) {
				os.Exit(1)
			}
			return nil
		},
	}
	return cmd
}
