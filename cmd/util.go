package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// readInput reads command input from --file, or stdin when no file is given.
func readInput(cmd *cobra.Command, file string) (string, error) {
	var r io.Reader
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return "", err
		}
		defer f.Close()
		r = f
	} else {
		r = cmd.InOrStdin()
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
