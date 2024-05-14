package cmd

import (
	"github.com/spf13/cobra"
)

func GetCheckCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "check",
		Aliases: []string{},
		Short:   "Check node setup",
		Run: func(cmd *cobra.Command, args []string) {
		},
	}

	return cmd
}

func init() {
	rootCmd.AddCommand(GetCheckCmd())
}
