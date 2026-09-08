package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand builds the rll command tree.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "rll",
		Short:         "RecallLoom Lite - lightweight file-based project memory",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.AddCommand(newInitCmd())
	root.AddCommand(newLogCmd())
	root.AddCommand(newWriteCmd())
	return root
}

// Execute runs the root command and exits non-zero on error.
func Execute(version string) {
	if err := NewRootCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
