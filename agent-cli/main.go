package main

// Built-in agent CLI. Each subcommand is one pipeline stage and is invoked
// both by the Temporal Activity (in the K8s Job) and locally via `act`.

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{Use: "agent", Short: "Test workflow agent"}

	root.AddCommand(
		investigateCmd(),
		genTestsCmd(),
		verifyCmd(),
		reviewSelfCmd(),
		submitCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
